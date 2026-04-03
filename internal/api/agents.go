package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"claude-agent-manager/internal/db"
)

type AgentRoutes struct {
	db               *db.DB
	sse              *SSEBroker
	updateLimiter    *RateLimiter
	fileLimiter      *RateLimiter
	webhookDispatch  func(string, map[string]interface{})
	onStatusChange   func(string, string)
	pushNotify       func(string, string, string)
}

func NewAgentRoutes(d *db.DB, sse *SSEBroker, webhookDispatch func(string, map[string]interface{}), onStatusChange func(string, string), pushNotify func(string, string, string)) *AgentRoutes {
	return &AgentRoutes{
		db:  d,
		sse: sse,
		updateLimiter: NewRateLimiter(60_000, 600, func(r *http.Request) string { // 600/min for streaming tool calls + thinking
			return r.PathValue("id")
		}),
		fileLimiter: NewRateLimiter(3_600_000, 20, func(r *http.Request) string {
			return r.PathValue("id")
		}),
		webhookDispatch: webhookDispatch,
		onStatusChange:  onStatusChange,
		pushNotify:      pushNotify,
	}
}

func (a *AgentRoutes) List(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 50, 100)
	cursor := r.URL.Query().Get("cursor")
	result := a.db.GetAllAgents(limit, cursor)
	writeJSON(w, http.StatusOK, result)
}

func (a *AgentRoutes) Analytics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.db.GetAnalytics())
}

func (a *AgentRoutes) Get(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *AgentRoutes) PostUpdate(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")

	var body map[string]interface{}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	updateType, _ := body["type"].(string)
	if updateType == "" {
		updateType = "status"
	}
	content := body["content"]
	summary, _ := body["summary"].(string)
	title, _ := body["title"].(string)
	status, _ := body["status"].(string)
	progress, _ := body["progress"].(float64)
	workspace, _ := body["workspace"].(string)
	cwd, _ := body["cwd"].(string)
	pidF, _ := body["pid"].(float64)

	// Create agent if doesn't exist
	existing := a.db.GetAgent(id)
	if existing == nil {
		agentTitle := title
		if agentTitle == "" {
			agentTitle = "Untitled Agent"
		}
		a.db.CreateAgent(id, agentTitle)

		// Link to project using env vars passed directly from the agent
		projectID, _ := body["project_id"].(string)
		agentRole, _ := body["role"].(string)
		parentAgentID, _ := body["parent_agent_id"].(string)

		if projectID != "" {
			a.db.Exec("UPDATE agents SET project_id = ?, role = ?, parent_agent_id = ? WHERE id = ?",
				nilIfEmpty(projectID), nilIfEmpty(agentRole), nilIfEmpty(parentAgentID), id)

			// If this is a PM (no parent), set it as the project's PM and deliver queued messages
			if parentAgentID == "" && agentRole == "PM" {
				a.db.UpdateProject(projectID, map[string]interface{}{"pm_agent_id": id})
			}

			slog.Info("Linked agent to project", "agentId", id, "projectId", projectID, "role", agentRole)
		}

		// Also try launch request matching for message delivery (PM prompts etc)
		a.linkAgentToProject(id)
	}

	// Update agent fields
	fields := map[string]interface{}{}
	if title != "" && existing != nil {
		fields["title"] = title
	}
	if status != "" {
		fields["status"] = status
	}
	if workspace != "" {
		fields["workspace"] = workspace
	}
	if cwd != "" {
		fields["cwd"] = cwd
	}
	if pidF > 0 {
		fields["pid"] = int(pidF)
	}
	if len(fields) > 0 {
		a.db.UpdateAgent(id, fields)
	}

	// Auto-unarchive
	if existing != nil {
		if s, _ := existing["status"].(string); s == "archived" {
			a.db.UpdateAgent(id, map[string]interface{}{"status": "active"})
		}
	}

	// Normalize content to JSON string
	var contentStr string
	switch v := content.(type) {
	case map[string]interface{}, []interface{}:
		b, _ := json.Marshal(v)
		contentStr = string(b)
	case string:
		// Check if already valid JSON
		var parsed interface{}
		if json.Unmarshal([]byte(v), &parsed) == nil {
			if _, ok := parsed.(map[string]interface{}); ok {
				contentStr = v
			} else {
				contentStr = wrapContent(updateType, v, progress)
			}
		} else {
			contentStr = wrapContent(updateType, v, progress)
		}
	default:
		contentStr = `{"text":""}`
	}

	var summaryPtr *string
	if summary != "" {
		summaryPtr = &summary
	}
	if err := a.db.AddUpdate(id, updateType, contentStr, summaryPtr); err != nil {
		slog.Error("Failed to add update", "agentId", id, "type", updateType, "err", err)
	}

	// Auto-acknowledge delivered messages
	a.db.AcknowledgeMessages(id)

	// Update metadata with projects/todos if provided
	projects, hasProjects := body["projects"]
	todos, hasTodos := body["todos"]
	if hasProjects || hasTodos {
		agent := a.db.GetAgent(id)
		if agent != nil {
			meta := map[string]interface{}{}
			if metaStr, ok := agent["metadata"].(string); ok && metaStr != "" {
				json.Unmarshal([]byte(metaStr), &meta)
			}
			if hasProjects {
				meta["projects"] = projects
			}
			if hasTodos {
				meta["todos"] = todos
			}
			metaJSON, _ := json.Marshal(meta)
			a.db.UpdateAgent(id, map[string]interface{}{"metadata": string(metaJSON)})
		}
	}

	updatedAgent := a.db.GetAgent(id)
	a.sse.Broadcast("agent-updated", updatedAgent)

	// Dispatch webhooks
	if status != "" {
		a.webhookDispatch("agent.status_changed", map[string]interface{}{"agent": updatedAgent, "details": map[string]string{"newStatus": status}})
		a.onStatusChange(id, status)
		if status == "completed" {
			a.webhookDispatch("agent.completed", map[string]interface{}{"agent": updatedAgent})
		} else if status == "waiting-for-input" {
			a.webhookDispatch("agent.waiting", map[string]interface{}{"agent": updatedAgent})
		}
	}
	if updateType == "error" {
		a.webhookDispatch("agent.error", map[string]interface{}{"agent": updatedAgent, "details": map[string]interface{}{"content": contentStr, "summary": summary}})
	}

	// Push notification
	agentTitle := "Untitled Agent"
	if updatedAgent != nil {
		if t, ok := updatedAgent["title"].(string); ok {
			agentTitle = t
		}
	}
	pushBody := summary
	if pushBody == "" {
		if s, ok := content.(string); ok {
			pushBody = s
		} else {
			b, _ := json.Marshal(content)
			pushBody = string(b)
		}
	}
	go a.pushNotify(agentTitle, pushBody, "/agent/"+id)

	pendingMessages, _ := a.db.GetPendingMessages(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "pendingMessages": pendingMessages})
}

func (a *AgentRoutes) linkAgentToProject(agentID string) {
	rows, err := a.db.Query("SELECT * FROM launch_requests WHERE status = 'completed' AND agent_id LIKE '{%' ORDER BY completed_at DESC LIMIT 5")
	if err != nil {
		return
	}
	defer rows.Close()
	reqs, _ := scanRowsHelper(rows)

	for _, req := range reqs {
		agentIDField, ok := req["agent_id"].(string)
		if !ok {
			continue
		}
		var meta map[string]interface{}
		if json.Unmarshal([]byte(agentIDField), &meta) != nil {
			continue
		}
		projectID, _ := meta["project_id"].(string)
		if projectID == "" {
			continue
		}

		role, _ := meta["role"].(string)
		parentAgentID, _ := meta["parent_agent_id"].(string)

		a.db.Exec("UPDATE agents SET project_id = ?, role = ?, parent_agent_id = ? WHERE id = ?",
			nilIfEmpty(projectID), nilIfEmpty(role), nilIfEmpty(parentAgentID), agentID)

		if role == "PM" && projectID != "" {
			a.db.UpdateProject(projectID, map[string]interface{}{"pm_agent_id": agentID})
			if pmPrompt, ok := meta["pm_prompt"].(string); ok && pmPrompt != "" {
				a.db.AddMessage(agentID, pmPrompt, "user", nil)
			}
			if userPrompt, ok := meta["user_prompt"].(string); ok && userPrompt != "" {
				a.db.AddMessage(agentID, userPrompt, "user", nil)
			}
		}

		// Sub-agent prompt
		if role != "PM" {
			if prompt, ok := meta["prompt"].(string); ok && prompt != "" {
				a.db.AddMessage(agentID, prompt, "user", nil)
			}
		}

		// Replace JSON metadata with real agent_id
		if id, ok := req["id"].(int64); ok {
			a.db.Exec("UPDATE launch_requests SET agent_id = ? WHERE id = ?", agentID, id)
		}

		slog.Info("Linked agent to project", "agentId", agentID, "projectId", projectID, "role", role)
		break
	}
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func wrapContent(updateType, content string, progress float64) string {
	switch updateType {
	case "progress":
		b, _ := json.Marshal(map[string]interface{}{"description": content, "percentage": progress})
		return string(b)
	case "error":
		b, _ := json.Marshal(map[string]interface{}{"message": content})
		return string(b)
	case "status":
		b, _ := json.Marshal(map[string]interface{}{"status": content})
		return string(b)
	default:
		b, _ := json.Marshal(map[string]interface{}{"text": content})
		return string(b)
	}
}

func (a *AgentRoutes) Patch(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	var body map[string]interface{}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	fields := map[string]interface{}{}
	if v, ok := body["title"].(string); ok {
		fields["title"] = v
	}
	if v, ok := body["status"].(string); ok {
		fields["status"] = v
	}
	if v, ok := body["project_id"].(string); ok {
		fields["project_id"] = v
	}
	if v, ok := body["role"].(string); ok {
		fields["role"] = v
	}
	if v, ok := body["parent_agent_id"].(string); ok {
		fields["parent_agent_id"] = v
	}
	if v, ok := body["cwd"].(string); ok {
		fields["cwd"] = v
	}
	if v, ok := body["metadata"]; ok {
		switch mt := v.(type) {
		case string:
			fields["metadata"] = mt
		default:
			b, _ := json.Marshal(mt)
			fields["metadata"] = string(b)
		}
	}
	if v, exists := body["poll_delay_until"]; exists {
		fields["poll_delay_until"] = v
	}
	if v, ok := body["workspace"].(string); ok {
		fields["workspace"] = v
	}
	if v, ok := body["cwd"].(string); ok {
		fields["cwd"] = v
	}
	if v, ok := body["pid"].(float64); ok {
		fields["pid"] = int(v)
	}

	a.db.UpdateAgent(id, fields)
	updated := a.db.GetAgent(id)
	a.sse.Broadcast("agent-updated", updated)
	writeJSON(w, http.StatusOK, updated)
}

func (a *AgentRoutes) MarkRead(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if a.db.GetAgent(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	a.db.UpdateAgent(id, map[string]interface{}{"last_read_at": now, "unread_update_count": 0})
	a.sse.Broadcast("agent-updated", a.db.GetAgent(id))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *AgentRoutes) Close(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		// Already closed/deleted — treat as success (idempotent)
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "already_closed": true})
		return
	}

	status, _ := agent["status"].(string)
	if status == "archived" || status == "completed" {
		// Already archived — treat as success
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "already_closed": true})
		return
	}

	a.db.UpdateAgent(id, map[string]interface{}{"status": "archived"})

	var terminated bool
	if pidF, ok := agent["pid"].(int64); ok && pidF > 0 {
		pid := int(pidF)
		a.db.CreateLaunchRequest("terminate", "", &id, &pid)
		terminated = true
	}

	a.sse.Broadcast("agent-updated", a.db.GetAgent(id))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "terminated": terminated, "pid": agent["pid"]})
}

func (a *AgentRoutes) Resume(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}
	s, _ := agent["status"].(string)
	if s != "archived" && s != "completed" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Agent is already %s, not archived/completed", s)})
		return
	}

	folderPath, _ := agent["cwd"].(string)
	lr := a.db.CreateLaunchRequest("resume", folderPath, &id, nil)
	a.db.UpdateAgent(id, map[string]interface{}{"status": "active"})
	a.sse.Broadcast("agent-updated", a.db.GetAgent(id))
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "launch_request_id": lr["id"]})
}

func (a *AgentRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if a.db.GetAgent(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	// Clean up files on disk
	for _, fp := range a.db.DeleteAgentFiles(id) {
		os.Remove(fp)
	}
	os.RemoveAll(filepath.Join("data", "files", id))

	a.db.DeleteAgent(id)
	a.sse.Broadcast("agent-deleted", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *AgentRoutes) GetUpdates(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if a.db.GetAgent(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}
	limit := parseIntQuery(r, "limit", 100, 200)
	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	writeJSON(w, http.StatusOK, a.db.GetUpdates(id, limit, before))
}

func (a *AgentRoutes) PostMessage(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	var body struct{ Content string `json:"content"` }
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	a.db.AddMessage(id, strings.TrimSpace(body.Content), "user", nil)
	a.sse.Broadcast("message-queued", map[string]interface{}{"agentId": id, "content": body.Content})
	a.webhookDispatch("message.received", map[string]interface{}{"agent": agent, "details": map[string]string{"content": body.Content}})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *AgentRoutes) GetMessages(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	statusFilter := r.URL.Query().Get("status")
	deliver := r.URL.Query().Get("deliver") == "true"

	if statusFilter == "pending" && deliver {
		// Heartbeat
		a.db.TouchAgentHeartbeat(id)

		// Auto-unarchive
		if s, _ := agent["status"].(string); s == "archived" {
			a.db.UpdateAgent(id, map[string]interface{}{"status": "active"})
			a.sse.Broadcast("agent-updated", a.db.GetAgent(id))
		}

		messages, _ := a.db.GetPendingMessages(id)
		agentData := a.db.GetAgent(id)
		if pdu, ok := agentData["poll_delay_until"].(string); ok && pdu != "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{"messages": messages, "poll_delay_until": pdu})
		} else {
			writeJSON(w, http.StatusOK, messages)
		}
		return
	}

	if statusFilter != "" {
		writeJSON(w, http.StatusOK, a.db.GetMessagesByStatus(id, statusFilter))
		return
	}

	limit := parseIntQuery(r, "limit", 100, 200)
	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	writeJSON(w, http.StatusOK, a.db.GetMessages(id, limit, before))
}

func (a *AgentRoutes) UploadFile(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if a.db.GetAgent(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	// Parse multipart (100MB max)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to parse multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	source := r.FormValue("source")
	if source == "" {
		source = "user"
	}
	description := r.FormValue("description")

	// Save to disk
	dir := filepath.Join("data", "files", id)
	os.MkdirAll(dir, 0755)
	prefix := strconv.FormatInt(time.Now().UnixMilli(), 36)
	diskPath := filepath.Join(dir, prefix+"_"+header.Filename)

	dst, err := os.Create(diskPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save file"})
		return
	}
	size, _ := io.Copy(dst, file)
	dst.Close()

	fileID, err := a.db.AddFile(id, header.Filename, header.Header.Get("Content-Type"), diskPath, size, source, description)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save file metadata"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
		"file": map[string]interface{}{
			"id": fileID, "filename": header.Filename, "source": source,
			"description": description, "mimetype": header.Header.Get("Content-Type"), "size": size,
		},
	})
}

func (a *AgentRoutes) ListFiles(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if a.db.GetAgent(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}
	limit := parseIntQuery(r, "limit", 50, 100)
	before, _ := strconv.Atoi(r.URL.Query().Get("before"))
	writeJSON(w, http.StatusOK, a.db.GetFilesMeta(id, limit, before))
}

func (a *AgentRoutes) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	fileID, _ := strconv.Atoi(pathParam(r, "fileId"))
	file := a.db.GetFile(id, fileID)
	if file == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "File not found"})
		return
	}

	fp, _ := file["file_path"].(string)
	if fp == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "File data not found on disk"})
		return
	}
	if _, err := os.Stat(fp); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "File data not found on disk"})
		return
	}

	mimetype, _ := file["mimetype"].(string)
	filename, _ := file["filename"].(string)
	w.Header().Set("Content-Type", mimetype)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	http.ServeFile(w, r, fp)
}

func (a *AgentRoutes) ExportPDF(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	agent := a.db.GetAgent(id)
	if agent == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Agent not found"})
		return
	}

	updates := a.db.GetUpdates(id, 10000, 0)
	messages := a.db.GetMessages(id, 10000, 0)
	files := a.db.GetFilesMeta(id, 1000, 0)

	payload := map[string]interface{}{
		"agent": agent, "updates": updates.Data, "messages": messages.Data, "files": files.Data,
	}

	pdfURL := os.Getenv("PDF_SERVICE_URL")
	if pdfURL == "" {
		pdfURL = "http://pdf-generator:8090"
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(pdfURL+"/generate/agent-report", "application/json", strings.NewReader(string(body)))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "PDF service unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errText, _ := io.ReadAll(resp.Body)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "PDF generation failed", "detail": string(errText)})
		return
	}

	pdfData, _ := io.ReadAll(resp.Body)
	agentTitle, _ := agent["title"].(string)
	if agentTitle == "" {
		agentTitle = "Agent"
	}
	filename := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, agentTitle) + "_Report.pdf"

	// Auto-upload
	dir := filepath.Join("data", "files", id)
	os.MkdirAll(dir, 0755)
	pdfPath := filepath.Join(dir, strconv.FormatInt(time.Now().UnixMilli(), 36)+"_"+filename)
	os.WriteFile(pdfPath, pdfData, 0644)
	a.db.AddFile(id, filename, "application/pdf", pdfPath, int64(len(pdfData)), "claude", "Auto-generated agent report")
	a.sse.Broadcast("agent-updated", a.db.GetAgent(id))

	go a.pushNotify(agentTitle, "PDF report ready for download", "/agent/"+id)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(pdfData)
}

func (a *AgentRoutes) Relay(w http.ResponseWriter, r *http.Request) {
	senderID := pathParam(r, "id")
	var body struct {
		TargetAgentID string `json:"target_agent_id"`
		Content       string `json:"content"`
	}
	if err := readJSON(r, &body); err != nil || body.TargetAgentID == "" || strings.TrimSpace(body.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_agent_id and content required"})
		return
	}

	if a.db.GetAgent(senderID) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Sender agent not found"})
		return
	}
	target := a.db.GetAgent(body.TargetAgentID)
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Target agent not found"})
		return
	}
	if senderID == body.TargetAgentID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Agent cannot send a message to itself"})
		return
	}

	a.db.AddMessage(body.TargetAgentID, strings.TrimSpace(body.Content), "agent", &senderID)
	a.sse.Broadcast("message-queued", map[string]interface{}{
		"agentId": body.TargetAgentID, "content": body.Content, "source": "agent", "sourceAgentId": senderID,
	})
	a.webhookDispatch("message.received", map[string]interface{}{
		"agent": target, "details": map[string]interface{}{"content": body.Content, "source": "agent", "sourceAgentId": senderID},
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// scanRowsHelper is a package-level helper to avoid import cycle
func scanRowsHelper(rows interface{ Next() bool; Columns() ([]string, error); Scan(dest ...interface{}) error }) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		result = append(result, row)
	}
	return result, nil
}
