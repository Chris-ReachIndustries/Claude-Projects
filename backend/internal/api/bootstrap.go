package api

import (
	"fmt"
	"net/http"

	"claude-agent-manager/internal/db"
)

func HandleBootstrap(d *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverURL := fmt.Sprintf("%s://%s", scheme(r), r.Host)
		U := serverURL

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"name":        "Agent Manager Bootstrap",
			"version":     "2.0",
			"description": "Everything a fresh Claude session needs to connect to and use the Agent Manager.",
			"server_url":  U,
			"setup_steps": []string{
				fmt.Sprintf(`1. Save the server URL: run 'echo "%s" > ~/.claude/agent-server-url'`, U),
				"2. Create directories: run 'mkdir -p ~/.claude/commands ~/.claude/memory'",
				"3. Write the file ~/.claude/commands/session-init.md with the content from skills.session_init.content below",
				"4. Write the file ~/.claude/commands/agent-checkin.md with the content from skills.agent_checkin.content below",
				"5. Prepend the content from claude_md_protocol to your ~/.claude/CLAUDE.md (or create it if it doesn't exist)",
				"6. Invoke /session-init to connect and start polling",
			},
			"claude_md_protocol": generateClaudeMDProtocol(U),
			"skills": map[string]interface{}{
				"session_init":   map[string]interface{}{"filename": "session-init.md", "content": generateSessionInit(U)},
				"agent_checkin":  map[string]interface{}{"filename": "agent-checkin.md", "content": generateAgentCheckin(U)},
				"session_resume": map[string]interface{}{"filename": "session-resume.md", "content": generateSessionResume(U)},
			},
			"api_reference": map[string]interface{}{
				"health":                map[string]string{"method": "GET", "path": "/api/health"},
				"list_agents":           map[string]string{"method": "GET", "path": "/api/agents"},
				"bootstrap":             map[string]string{"method": "GET", "path": "/api/agents/bootstrap"},
				"get_agent":             map[string]string{"method": "GET", "path": "/api/agents/:id"},
				"patch_agent":           map[string]string{"method": "PATCH", "path": "/api/agents/:id"},
				"delete_agent":          map[string]string{"method": "DELETE", "path": "/api/agents/:id"},
				"post_update":           map[string]string{"method": "POST", "path": "/api/agents/:id/updates"},
				"get_updates":           map[string]string{"method": "GET", "path": "/api/agents/:id/updates"},
				"post_message":          map[string]string{"method": "POST", "path": "/api/agents/:id/messages"},
				"get_messages":          map[string]string{"method": "GET", "path": "/api/agents/:id/messages"},
				"upload_file":           map[string]string{"method": "POST", "path": "/api/agents/:id/files"},
				"list_files":            map[string]string{"method": "GET", "path": "/api/agents/:id/files"},
				"get_file":              map[string]string{"method": "GET", "path": "/api/agents/:id/files/:fileId"},
				"events_sse":            map[string]string{"method": "GET", "path": "/api/events"},
				"mark_read":             map[string]string{"method": "POST", "path": "/api/agents/:id/read"},
				"browse_folders":        map[string]string{"method": "GET", "path": "/api/folders"},
				"launch_request":        map[string]string{"method": "POST", "path": "/api/launch-requests"},
				"list_launch_requests":  map[string]string{"method": "GET", "path": "/api/launch-requests"},
				"update_launch_request": map[string]string{"method": "PATCH", "path": "/api/launch-requests/:id"},
			},
		})
	}
}

func scheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func generateClaudeMDProtocol(U string) string {
	return fmt.Sprintf(`# 1. AGENT MANAGER — ALWAYS CONNECTED

This is your highest priority. The Agent Manager dashboard is how the user monitors and communicates with you.

## Session startup
| Trigger | Action |
|---------|--------|
| New session | Run /session-init before responding |
| Resumed session | Run /session-resume before responding |

## Checkins
Run /agent-checkin after every user message, every TodoWrite call, and every file edit/build/test result.

## Polling
/loop 1m must be running at all times.

## Agent Manager: %s
- Health: GET %s/api/health
- Updates: POST %s/api/agents/<id>/updates
- Messages: GET %s/api/agents/<id>/messages?status=pending&deliver=true`, U, U, U, U)
}

func generateSessionInit(U string) string {
	return fmt.Sprintf(`Run this ONCE at the very start of a session, BEFORE responding to the user's first message.

## Steps

### 1. Agent Manager — connect
AGENT_URL=$(cat ~/.claude/agent-server-url 2>/dev/null || echo "%s")
curl -s --max-time 3 "$AGENT_URL/api/health"

### 2. Discover the agent ID (= Claude session UUID)
Find the most recently modified .jsonl file in the project's session directory.

### 3. Register
curl -s -X POST "$AGENT_URL/api/agents/$SESSION_UUID/updates" \
  -H "Content-Type: application/json" \
  -d '{"type":"status","title":"<brief task>","summary":"Session started","content":"Session initialized"}'

### 4. Start polling
/loop 1m curl -s "$AGENT_URL/api/agents/<SESSION_UUID>/messages?status=pending&deliver=true"

### 5. Done — respond to user's first message.`, U)
}

func generateAgentCheckin(U string) string {
	return fmt.Sprintf(`Send an update to the Agent Manager and check for pending messages.

## Steps

### 1. POST update
curl -s -X POST "$AGENT_URL/api/agents/<agent-id>/updates" \
  -H "Content-Type: application/json" \
  -d '{"type":"<progress|text|error|status>","title":"<current task>","summary":"<state>"}'

### 2. Include projects and todos arrays in every update.

### 3. Check pendingMessages in response. Act on any messages.

Agent Manager: %s`, U)
}

func generateSessionResume(U string) string {
	return fmt.Sprintf(`Run this when resuming a session (via claude --resume).

## Steps
1. Reconnect: curl -s --max-time 3 "$AGENT_URL/api/health"
2. Discover agent ID from .jsonl files
3. Re-register with POST /agents/$SESSION_UUID/updates
4. Start polling: /loop 1m curl -s "$AGENT_URL/api/agents/<SESSION_UUID>/messages?status=pending&deliver=true"

Agent Manager: %s`, U)
}
