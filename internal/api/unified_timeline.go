package api

import (
	"net/http"
	"sort"

	"claude-agent-manager/internal/db"
)

// HandleUnifiedTimeline returns a merged timeline for a project:
// project updates + all agent updates, sorted chronologically.
func HandleUnifiedTimeline(d *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := pathParam(r, "id")
		project := d.GetProject(projectID)
		if project == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Project not found"})
			return
		}

		limit := parseIntQuery(r, "limit", 50, 200)

		type timelineEntry struct {
			ID        interface{} `json:"id"`
			Source    string      `json:"source"`     // "project", "agent"
			AgentID   string      `json:"agent_id,omitempty"`
			AgentRole string      `json:"agent_role,omitempty"`
			AgentName string      `json:"agent_name,omitempty"`
			Type      string      `json:"type"`       // update type
			Summary   string      `json:"summary"`
			Content   string      `json:"content"`
			Timestamp string      `json:"timestamp"`
		}

		var entries []timelineEntry

		// 1. Project updates
		projectUpdates := d.GetProjectUpdates(projectID, 100, 0)
		for _, u := range projectUpdates.Data {
			entries = append(entries, timelineEntry{
				ID:        u["id"],
				Source:    "project",
				Type:      strVal(u["type"]),
				Summary:   strVal(u["type"]) + ": " + truncate(strVal(u["content"]), 100),
				Content:   strVal(u["content"]),
				Timestamp: strVal(u["timestamp"]),
			})
		}

		// 2. Agent updates for all project agents
		agents := d.GetProjectAgents(projectID)
		for _, agent := range agents {
			agentID := strVal(agent["id"])
			agentRole := strVal(agent["role"])
			agentName := strVal(agent["title"])
			if agentRole == "" {
				agentRole = "Agent"
			}
			if agentName == "" {
				agentName = "Unnamed"
			}

			agentUpdates := d.GetUpdates(agentID, 50, 0)
			for _, u := range agentUpdates.Data {
				summary := strVal(u["summary"])
				// Skip noise entries
				if summary == "Polling for messages" || summary == "" {
					continue
				}
				entries = append(entries, timelineEntry{
					ID:        u["id"],
					Source:    "agent",
					AgentID:   agentID,
					AgentRole: agentRole,
					AgentName: agentName,
					Type:      strVal(u["type"]),
					Summary:   summary,
					Content:   strVal(u["content"]),
					Timestamp: strVal(u["timestamp"]),
				})
			}
		}

		// Sort by timestamp descending (newest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timestamp > entries[j].Timestamp
		})

		// Limit
		if len(entries) > limit {
			entries = entries[:limit]
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"entries":     entries,
			"total_count": len(entries),
		})
	}
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
