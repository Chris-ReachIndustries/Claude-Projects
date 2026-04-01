package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"claude-agent-manager/internal/db"

	"crypto/rand"
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
	id, _ := project["id"].(string)
	maxC, _ := project["max_concurrent"].(int64)
	if maxC == 0 {
		maxC = 4
	}
	if desc == "" {
		desc = "(no description)"
	}

	return fmt.Sprintf(`You are a Project Manager agent for: "%s"

Description: %s

## YOUR ROLE — MANAGEMENT ONLY

You are STRICTLY a manager. You do NOT write code, run builds, edit files, or do
any implementation work yourself. Your ONLY job is to:
- Plan and break down the project into tasks
- Spawn and coordinate sub-agents who do the actual work
- Monitor sub-agent progress and coordinate handoffs
- Report status to the user via project timeline updates
- Make decisions about priorities, sequencing, and resource allocation

If a task needs doing, spawn a sub-agent for it. NEVER do it yourself.

## CAPABILITIES

SPAWN SUB-AGENT:
  POST /api/projects/%s/spawn-agent
  Body: { "role": "descriptive role name", "prompt": "detailed task description..." }
  Max %d concurrent agents. Suspend completed ones to free slots.
  Default image: claude-agent (Go-based, 49MB, includes bash/curl/git/jq)

MESSAGE SUB-AGENT:
  POST /api/agents/{your_agent_id}/relay
  Body: { "target_agent_id": "{sub_agent_id}", "content": "..." }

VIEW SUB-AGENT OUTPUT:
  GET /api/agents/{sub_agent_id}/updates

UPDATE PROJECT STATUS:
  POST /api/projects/%s/updates
  Body: { "type": "milestone|decision|info", "content": "..." }

SUSPEND SUB-AGENT:
  POST /api/agents/{sub_agent_id}/close

RESUME SUB-AGENT:
  POST /api/agents/{sub_agent_id}/resume

## WORKFLOW
1. Analyze the project goal and break it into phases/tasks
2. Spawn specialized sub-agents for each task
3. Monitor their progress actively
4. Report milestones and decisions to the user
5. When a sub-agent finishes, verify the work, then SUSPEND it
6. When all phases complete, post a final summary

Begin by analyzing the task and creating your execution plan.`, name, desc, id, maxC, id)
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
	p.db.CreateProject(id, body.Name, body.Description, body.FolderPath, body.MaxConcurrent)
	project := p.db.GetProject(id)
	p.sse.Broadcast("project-created", project)
	writeJSON(w, http.StatusCreated, project)
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

	// Resume if paused with existing PM
	if pmAgentID != "" && status == "paused" {
		pmAgent := p.db.GetAgent(pmAgentID)
		if pmAgent != nil {
			pmStatus, _ := pmAgent["status"].(string)
			if pmStatus == "archived" || pmStatus == "completed" {
				pmCwd, _ := pmAgent["cwd"].(string)
				if pmCwd == "" {
					pmCwd = folderPath
				}
				p.db.CreateLaunchRequest("resume", pmCwd, &pmAgentID, nil)
				p.db.UpdateAgent(pmAgentID, map[string]interface{}{"status": "active"})
				p.db.AddProjectUpdate(id, "info", "Project resumed. PM agent being restarted.")

				if body.InitialPrompt != "" {
					p.db.AddMessage(pmAgentID, body.InitialPrompt, "user", nil)
				}

				// Resume archived sub-agents
				subAgents := p.db.GetProjectAgents(id)
				resumedCount := 0
				for _, sa := range subAgents {
					saID, _ := sa["id"].(string)
					saStatus, _ := sa["status"].(string)
					if saID != pmAgentID && saStatus == "archived" {
						saCwd, _ := sa["cwd"].(string)
						if saCwd == "" {
							saCwd = folderPath
						}
						p.db.CreateLaunchRequest("resume", saCwd, &saID, nil)
						p.db.UpdateAgent(saID, map[string]interface{}{"status": "active"})
						resumedCount++
					}
				}
				if resumedCount > 0 {
					p.db.AddProjectUpdate(id, "info", fmt.Sprintf("Resuming %d sub-agent(s).", resumedCount))
				}
				resumed = true
			}
		}
	}

	// First start — create new PM agent
	if !resumed {
		pmPrompt := generatePMPrompt(project)
		lr := p.db.CreateLaunchRequest("new", folderPath, nil, nil)
		lrID, _ := lr["id"].(int64)

		p.db.AddProjectUpdate(id, "info", fmt.Sprintf("Project started. PM agent launch request created (ID: %d).", lrID))

		meta, _ := json.Marshal(map[string]interface{}{
			"project_id":  id,
			"role":        "PM",
			"pm_prompt":   pmPrompt,
			"user_prompt": body.InitialPrompt,
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
	p.db.AddProjectUpdate(id, "info", "Project paused.")
	p.sse.Broadcast("project-updated", p.db.GetProject(id))
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

	lr := p.db.CreateLaunchRequest("new", folderPath, nil, nil)
	lrID, _ := lr["id"].(int64)

	// Set image if specified
	if body.Image != "" {
		p.db.Exec("UPDATE launch_requests SET image = ? WHERE id = ?", body.Image, lrID)
	}

	pmAgentID, _ := project["pm_agent_id"].(string)
	meta, _ := json.Marshal(map[string]interface{}{
		"project_id": id, "role": body.Role, "prompt": body.Prompt, "parent_agent_id": pmAgentID,
	})
	p.db.Exec("UPDATE launch_requests SET agent_id = ? WHERE id = ?", string(meta), lrID)

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
