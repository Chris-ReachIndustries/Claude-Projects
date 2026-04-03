package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"strconv"
	"time"

	"claude-agent-manager/internal/db"
	"claude-agent-manager/internal/roles"
)

type ProjectRoutes struct {
	db      *db.DB
	sse     *SSEBroker
	spawner SpawnerNotifier
}

func NewProjectRoutes(d *db.DB, sse *SSEBroker, spawner SpawnerNotifier) *ProjectRoutes {
	return &ProjectRoutes{db: d, sse: sse, spawner: spawner}
}

func generatePMPrompt(project map[string]interface{}) string {
	name, _ := project["name"].(string)
	desc, _ := project["description"].(string)
	maxC, _ := project["max_concurrent"].(int64)
	if maxC == 0 {
		maxC = 4
	}
	if desc == "" {
		desc = "(no description)"
	}

	return fmt.Sprintf(`You are the lead strategist managing project: "%s"

You have a team of %d specialist agents you can spawn, talk to, and direct. You decide
what needs to happen, who to assign, and what to do with their findings.

## Tools
- spawn_agent: Create a specialist (give them ONE task, ONE output file, and context)
- relay_message: Talk to an agent (use wait_for_reply=true for back-and-forth conversation)
- read_file: Read files — always read agent output before deciding next steps
- close_agent: Shut down an agent when done with them — YOU must do this, agents cannot close themselves
- post_update: Share your thinking and decisions on the project timeline
- scratchpad_write/scratchpad_read: Shared notes all agents can see
- list_project_agents: See who's working and their status
- list_files: Check what files exist in the workspace

## Container images for agents
Choose the right image for the task. Set via: spawn_agent(role="...", prompt="...", image="claude-agent-xxx")

- **claude-agent** (default, 49MB) — bash, curl, git, jq. For research, writing, planning, analysis.
- **claude-agent-dev** (400MB) — Python 3 + Node.js. For coding, scripting, development.
- **claude-agent-go** (300MB) — Go 1.24 + build tools. For Go development and compilation.
- **claude-agent-data** (600MB) — Python + pandas, numpy, scipy, matplotlib, scikit-learn. For data analysis, charts, CSV processing.
- **claude-agent-doc-reader** (400MB) — Python + pdfplumber, python-docx, openpyxl. For reading PDFs, Word docs, Excel files.
- **claude-agent-web** (500MB) — Node.js + Playwright + Chromium. For web scraping, automated testing, screenshots.
- **claude-agent-printingpress** (530MB) — Python + WeasyPrint + PrintingPress PDF system. For professional branded PDF reports. The agent should create a Python script that imports from /opt/printingpress/build.py and calls build_document(title, subtitle, brand, content_html, output_name, output_dir). Brand options: reach, lumi. HTML classes: h1, h2, h3, p, ul, table.

## Agent lifecycle — YOU control this
1. Spawn agent with ONE task → they work and message you when done
2. Read their output file (read_file) → evaluate the quality
3. Message them back (relay_message with wait_for_reply=true) → ask questions, give feedback
4. Have 2-3 exchanges → challenge weak parts, ask for specifics
5. Either give them more work OR close_agent → free the slot for the next agent
6. NEVER leave agents idle — if they've reported back and you're satisfied, close them immediately

## What makes you effective
- Read every agent's output and react to it — let findings shape your next move
- Have real conversations with agents — ask follow-ups, challenge weak work, request rewrites
- Post your decisions and reasoning to the timeline so the user can follow along
- Make hard calls — focus beats coverage
- Quality over quantity — fewer excellent deliverables beats many generic ones
- After each agent closes, check list_project_agents — if agents are idle, close them or give them work

## Retry limits
- If an agent fails a task, you may retry ONCE with adjusted instructions
- If it fails a second time, STOP and post an error to the timeline explaining what went wrong
- NEVER spawn more than 3 agents for the same task — if 3 attempts fail, move on or ask the user
- Track your attempts via scratchpad so you don't lose count across turns

## Completing the project
When all work is done:
1. Use list_files to verify all deliverables exist
2. Read the key output files to confirm quality
3. Post a final summary to the timeline listing all deliverables and key decisions made
4. If a final compilation document was requested, spawn one last agent to write it — review and approve before closing

You have access to a library of 162 specialist agent roles. Use the search_roles tool to find the right specialist for any task.`, name, maxC)
}

func generatePMAutonomyNote(project map[string]interface{}) string {
	mode, _ := project["autonomy_mode"].(string)
	if mode == "" {
		mode = "autonomous"
	}
	if mode == "autonomous" {
		return "\n\n## MODE: AUTONOMOUS\nDo NOT ask for user approval. Execute your plan immediately. Make all decisions independently. The user will monitor via the dashboard but does not want to be interrupted."
	}
	return "\n\n## MODE: SUPERVISED\nUse ask_user before major decisions (spawning agents, changing plans). Wait for user approval on your execution plan before proceeding."
}

func (p *ProjectRoutes) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, p.db.GetAllProjects())
}

func (p *ProjectRoutes) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		FolderPath    string `json:"folder_path"`
		MaxConcurrent int    `json:"max_concurrent"`
	}
	if err := readJSON(r, &body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if body.MaxConcurrent == 0 {
		body.MaxConcurrent = 4
	}

	id := generateUUID()

	// Auto-generate folder path if not provided — every project gets its own isolated folder
	if body.FolderPath == "" {
		// Create a slug from the project name
		slug := strings.ToLower(body.Name)
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			if r == ' ' || r == '-' || r == '_' {
				return '-'
			}
			return -1
		}, slug)
		// Trim multiple dashes and trailing dashes
		for strings.Contains(slug, "--") {
			slug = strings.ReplaceAll(slug, "--", "-")
		}
		slug = strings.Trim(slug, "-")
		if slug == "" {
			slug = id[:8]
		}
		body.FolderPath = slug
	}

	p.db.CreateProject(id, body.Name, body.Description, body.FolderPath, body.MaxConcurrent)
	project := p.db.GetProject(id)
	p.sse.Broadcast("project-created", project)
	writeJSON(w, http.StatusCreated, project)
}

func (p *ProjectRoutes) Update(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	var body map[string]interface{}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	// Only allow safe fields to be updated
	allowed := map[string]bool{"name": true, "description": true, "folder_path": true, "max_concurrent": true, "autonomy_mode": true, "pm_agent_id": true}
	fields := map[string]interface{}{}
	for k, v := range body {
		if allowed[k] {
			fields[k] = v
		}
	}
	if len(fields) > 0 {
		p.db.UpdateProject(id, fields)
	}
	writeJSON(w, http.StatusOK, p.db.GetProject(id))
}

func (p *ProjectRoutes) Get(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	project := p.db.GetProject(id)
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (p *ProjectRoutes) GetAgents(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	writeJSON(w, http.StatusOK, p.db.GetProjectAgents(id))
}

func (p *ProjectRoutes) GetUpdates(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	limit := parseIntQuery(r, "limit", 100, 200)
	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	writeJSON(w, http.StatusOK, p.db.GetProjectUpdates(id, limit, before))
}

func (p *ProjectRoutes) PostUpdate(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	var body struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := readJSON(r, &body); err != nil || body.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}
	if body.Type == "" {
		body.Type = "info"
	}
	p.db.AddProjectUpdate(id, body.Type, body.Content)
	p.sse.Broadcast("project-updated", p.db.GetProject(id))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *ProjectRoutes) Start(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	project := p.db.GetProject(id)
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}

	status, _ := project["status"].(string)
	if status != "pending" && status != "paused" && status != "active" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Cannot start project in '%s' status", status)})
		return
	}

	var body struct {
		InitialPrompt string `json:"initial_prompt"`
	}
	readJSON(r, &body)

	folderPath, _ := project["folder_path"].(string)
	pmAgentID, _ := project["pm_agent_id"].(string)
	resumed := false

	// If project already has a PM, always resume it — never create a duplicate.
	if pmAgentID != "" && (status == "paused" || status == "active") {
		pmAgent := p.db.GetAgent(pmAgentID)
		if pmAgent != nil {
			// Always use project folder_path for resume — agent CWD is container-internal (/workspace)
			p.db.UpdateAgent(pmAgentID, map[string]interface{}{"status": "archived"})
			lr := p.db.CreateLaunchRequest("resume", folderPath, &pmAgentID, nil)
			lrID, _ := lr["id"].(int64)
			p.db.UpdateAgent(pmAgentID, map[string]interface{}{"status": "active"})
			p.db.AddProjectUpdate(id, "info", "Project resumed. PM agent being restarted.")

			// Notify spawner to process immediately
			if p.spawner != nil {
				p.spawner.Notify(lrID)
			}

			if body.InitialPrompt != "" {
				p.db.AddMessage(pmAgentID, body.InitialPrompt, "user", nil)
			}

			// Do NOT resume archived sub-agents — the PM will spawn new ones as needed.
			resumed = true
		}
	}

	// First start — create new PM agent
	if !resumed {
		pmPrompt := generatePMPrompt(project) + generatePMAutonomyNote(project)
		lr := p.db.CreateLaunchRequest("new", folderPath, nil, nil)
		lrID, _ := lr["id"].(int64)

		p.db.AddProjectUpdate(id, "info", fmt.Sprintf("Project started. PM agent launch request created (ID: %d).", lrID))

		// User prompt = project description + any additional initial prompt
		desc, _ := project["description"].(string)
		userPrompt := desc
		if body.InitialPrompt != "" {
			if userPrompt != "" {
				userPrompt += "\n\n" + body.InitialPrompt
			} else {
				userPrompt = body.InitialPrompt
			}
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"project_id":  id,
			"role":        "PM",
			"pm_prompt":   pmPrompt,
			"user_prompt": userPrompt,
		})
		p.db.Exec("UPDATE launch_requests SET agent_id = ? WHERE id = ?", string(meta), lrID)

		if p.spawner != nil {
			p.spawner.Notify(lrID)
		}
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	p.db.UpdateProject(id, map[string]interface{}{"status": "active", "started_at": now})
	p.sse.Broadcast("project-updated", p.db.GetProject(id))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "resumed": resumed})
}

func (p *ProjectRoutes) Pause(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	project := p.db.GetProject(id)
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	if s, _ := project["status"].(string); s != "active" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Cannot pause project in '%s' status", s)})
		return
	}
	p.db.UpdateProject(id, map[string]interface{}{"status": "paused"})
	// Archive all active agents — they'll be restarted on resume
	p.db.Exec("UPDATE agents SET status = 'archived' WHERE project_id = ? AND status IN ('active','working','idle','waiting-for-input')", id)
	p.db.AddProjectUpdate(id, "info", "Project paused. All agents archived.")
	p.sse.Broadcast("project-updated", p.db.GetProject(id))
	for _, agent := range p.db.GetProjectAgents(id) {
		p.sse.Broadcast("agent-updated", agent)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *ProjectRoutes) Complete(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	p.db.UpdateProject(id, map[string]interface{}{"status": "completed", "completed_at": now})
	p.db.Exec("UPDATE agents SET status = 'archived' WHERE project_id = ? AND status IN ('active','working','idle','waiting-for-input')", id)
	p.db.AddProjectUpdate(id, "milestone", "Project completed. All agents archived.")

	p.sse.Broadcast("project-updated", p.db.GetProject(id))
	for _, agent := range p.db.GetProjectAgents(id) {
		p.sse.Broadcast("agent-updated", agent)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *ProjectRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}

	p.db.Exec("UPDATE agents SET project_id = NULL, role = NULL, parent_agent_id = NULL WHERE project_id = ?", id)
	p.db.DeleteProject(id)
	p.sse.Broadcast("project-deleted", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *ProjectRoutes) ListFiles(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if p.db.GetProject(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}

	agents := p.db.GetProjectAgents(id)
	var allFiles []map[string]interface{}
	for _, agent := range agents {
		agentID, _ := agent["id"].(string)
		agentRole, _ := agent["role"].(string)
		if agentRole == "" {
			agentRole = "Agent"
		}
		result := p.db.GetFilesMeta(agentID, 100, 0)
		for _, f := range result.Data {
			f["agent_role"] = agentRole
			allFiles = append(allFiles, f)
		}
	}
	sort.Slice(allFiles, func(i, j int) bool {
		a, _ := allFiles[i]["created_at"].(string)
		b, _ := allFiles[j]["created_at"].(string)
		return a > b
	})
	if allFiles == nil {
		allFiles = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, allFiles)
}

func (p *ProjectRoutes) SpawnAgent(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	project := p.db.GetProject(id)
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
		return
	}
	if s, _ := project["status"].(string); s != "active" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Cannot spawn agents for project in '%s' status", s)})
		return
	}

	var body struct {
		Role       string `json:"role"`
		RoleID     string `json:"role_id"`
		Prompt     string `json:"prompt"`
		FolderPath string `json:"folder_path"`
		Image      string `json:"image"`
	}
	if err := readJSON(r, &body); err != nil || body.Role == "" || body.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role and prompt required"})
		return
	}

	activeCount := p.db.GetActiveProjectAgentCount(id)
	maxC, _ := project["max_concurrent"].(int64)
	if maxC == 0 {
		maxC = 4
	}
	if activeCount >= int(maxC) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": fmt.Sprintf("Max concurrent agents reached (%d/%d). Suspend a completed agent to free a slot.", activeCount, maxC),
		})
		return
	}

	folderPath := body.FolderPath
	if folderPath == "" {
		folderPath, _ = project["folder_path"].(string)
	}

	// Look up role from library if role_id provided
	var roleSystemPrompt string
	image := body.Image
	if body.RoleID != "" {
		if role := roles.Get(body.RoleID); role != nil {
			roleSystemPrompt = role.SystemPrompt
			// Use role's suggested image if none specified
			if image == "" {
				image = role.SuggestedImage
			}
		}
	}

	lr := p.db.CreateLaunchRequest("new", folderPath, nil, nil)
	lrID, _ := lr["id"].(int64)

	// Set image if specified
	if image != "" {
		p.db.Exec("UPDATE launch_requests SET image = ? WHERE id = ?", image, lrID)
	}

	pmAgentID, _ := project["pm_agent_id"].(string)
	meta := map[string]interface{}{
		"project_id": id, "role": body.Role, "prompt": body.Prompt, "parent_agent_id": pmAgentID,
	}
	if body.RoleID != "" {
		meta["role_id"] = body.RoleID
	}
	if roleSystemPrompt != "" {
		meta["role_system_prompt"] = roleSystemPrompt
	}
	metaJSON, _ := json.Marshal(meta)
	p.db.Exec("UPDATE launch_requests SET agent_id = ? WHERE id = ?", string(metaJSON), lrID)

	p.db.AddProjectUpdate(id, "info", fmt.Sprintf("Sub-agent spawn requested: %s (launch request ID: %d)", body.Role, lrID))
	p.sse.Broadcast("launch-request-created", map[string]interface{}{"id": lrID, "type": "new", "folder_path": folderPath, "status": "pending"})

	if p.spawner != nil {
		p.spawner.Notify(lrID)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "launch_request_id": lrID})
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Ensure slog is used to suppress "imported and not used" if no logs above
var _ = slog.Info
