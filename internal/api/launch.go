package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claude-agent-manager/internal/db"
)

type LaunchRoutes struct {
	db      *db.DB
	sse     *SSEBroker
	limiter *RateLimiter
	spawner SpawnerNotifier
}

func NewLaunchRoutes(d *db.DB, sse *SSEBroker, spawner SpawnerNotifier) *LaunchRoutes {
	return &LaunchRoutes{
		db:      d,
		sse:     sse,
		spawner: spawner,
		limiter: NewRateLimiter(300_000, 10, func(r *http.Request) string {
			return r.RemoteAddr
		}),
	}
}

func (l *LaunchRoutes) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type          string `json:"type"`
		FolderPath    string `json:"folder_path"`
		ResumeAgentID string `json:"resume_agent_id"`
		TargetPID     *int   `json:"target_pid"`
		Image         string `json:"image"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}
	if body.Type == "" {
		body.Type = "new"
	}
	if body.Type == "new" && body.FolderPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path is required for new agent launches"})
		return
	}
	if body.Type == "resume" && body.ResumeAgentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resume_agent_id is required for resume launches"})
		return
	}
	if body.Type == "terminate" && body.ResumeAgentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resume_agent_id (target agent) is required for terminate requests"})
		return
	}

	var raid *string
	if body.ResumeAgentID != "" {
		raid = &body.ResumeAgentID
	}
	request := l.db.CreateLaunchRequest(body.Type, body.FolderPath, raid, body.TargetPID)

	// Save image if specified
	if body.Image != "" {
		if id, ok := request["id"].(int64); ok {
			l.db.Exec("UPDATE launch_requests SET image = ? WHERE id = ?", body.Image, id)
		}
	}

	l.sse.Broadcast("launch-request-created", request)

	// Notify spawner to process immediately
	if id, ok := request["id"].(int64); ok && l.spawner != nil {
		l.spawner.Notify(id)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "request": request})
}

func (l *LaunchRoutes) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" {
		writeJSON(w, http.StatusOK, l.db.GetLaunchRequestsByStatus(status))
		return
	}
	pending := l.db.GetLaunchRequestsByStatus("pending")
	claimed := l.db.GetLaunchRequestsByStatus("claimed")
	all := append(pending, claimed...)
	writeJSON(w, http.StatusOK, all)
}

func (l *LaunchRoutes) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(pathParam(r, "id"))

	existing := l.db.GetLaunchRequest(id)
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Launch request not found"})
		return
	}

	var body struct {
		Status  string `json:"status"`
		AgentID string `json:"agent_id"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	// Check if existing agent_id contains project metadata
	var projectMeta map[string]interface{}
	if existingAgentID, ok := existing["agent_id"].(string); ok && existingAgentID != "" {
		var parsed map[string]interface{}
		if json.Unmarshal([]byte(existingAgentID), &parsed) == nil {
			if _, hasProject := parsed["project_id"]; hasProject {
				projectMeta = parsed
			}
		}
	}

	fields := map[string]interface{}{}
	if body.Status != "" {
		fields["status"] = body.Status
	}
	if body.AgentID != "" {
		fields["agent_id"] = body.AgentID
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if body.Status == "claimed" {
		fields["claimed_at"] = now
	}
	if body.Status == "completed" || body.Status == "failed" {
		fields["completed_at"] = now
	}
	l.db.UpdateLaunchRequest(id, fields)

	// Link agent to project if metadata present
	if projectMeta != nil && body.AgentID != "" && body.Status == "completed" {
		l.linkAgentToProject(body.AgentID, projectMeta)
	}

	updated := l.db.GetLaunchRequest(id)
	l.sse.Broadcast("launch-request-updated", updated)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "request": updated})
}

func (l *LaunchRoutes) linkAgentToProject(agentID string, meta map[string]interface{}) {
	agent := l.db.GetAgent(agentID)
	if agent == nil {
		return
	}

	projectID, _ := meta["project_id"].(string)
	role, _ := meta["role"].(string)
	parentAgentID, _ := meta["parent_agent_id"].(string)

	l.db.Exec("UPDATE agents SET project_id = ?, role = ?, parent_agent_id = ? WHERE id = ?",
		nilIfEmpty(projectID), nilIfEmpty(role), nilIfEmpty(parentAgentID), agentID)

	if role == "PM" && projectID != "" {
		l.db.UpdateProject(projectID, map[string]interface{}{"pm_agent_id": agentID})
		if pmPrompt, ok := meta["pm_prompt"].(string); ok && pmPrompt != "" {
			l.db.AddMessage(agentID, pmPrompt, "user", nil)
			slog.Info("Sent PM system prompt", "agentId", agentID)
		}
		if userPrompt, ok := meta["user_prompt"].(string); ok && userPrompt != "" {
			l.db.AddMessage(agentID, userPrompt, "user", nil)
			slog.Info("Sent user initial prompt", "agentId", agentID)
		}
	}

	if role != "PM" {
		if prompt, ok := meta["prompt"].(string); ok && prompt != "" {
			l.db.AddMessage(agentID, prompt, "user", nil)
			slog.Info("Sent sub-agent task prompt", "agentId", agentID, "role", role)
		}
	}

	slog.Info("Linked agent to project from launch request", "agentId", agentID, "projectId", projectID)
}
