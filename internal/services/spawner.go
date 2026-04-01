package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"claude-agent-manager/internal/db"
)

// Spawner processes launch requests by spawning Docker containers.
// Routes push request IDs onto the channel; the spawner reacts instantly.
type Spawner struct {
	db       *db.DB
	Requests chan int64
	sse      interface{ Broadcast(string, interface{}) }
}

func NewSpawner(d *db.DB, sse interface{ Broadcast(string, interface{}) }) *Spawner {
	return &Spawner{
		db:       d,
		Requests: make(chan int64, 100),
		sse:      sse,
	}
}

// Notify pushes a launch request ID for processing.
func (s *Spawner) Notify(requestID int64) {
	select {
	case s.Requests <- requestID:
	default:
		slog.Warn("Spawner channel full, dropping request", "requestId", requestID)
	}
}

// Start begins the spawner goroutine.
func (s *Spawner) Start() {
	slog.Info("Container spawner started")

	// Process any pending requests from before startup
	go func() {
		time.Sleep(2 * time.Second)
		pending := s.db.GetLaunchRequestsByStatus("pending")
		for _, req := range pending {
			if id, ok := req["id"].(int64); ok {
				s.Notify(id)
			}
		}
	}()

	// Main processing loop
	go func() {
		for reqID := range s.Requests {
			s.processRequest(reqID)
		}
	}()
}

func (s *Spawner) processRequest(reqID int64) {
	req := s.db.GetLaunchRequest(int(reqID))
	if req == nil {
		return
	}

	status, _ := req["status"].(string)
	if status != "pending" {
		return // already claimed or processed
	}

	reqType, _ := req["type"].(string)
	folderPath, _ := req["folder_path"].(string)
	resumeAgentID, _ := req["resume_agent_id"].(string)

	// Extract role/title from project metadata if present
	agentTitle := "Container Agent"
	if agentIDField, ok := req["agent_id"].(string); ok && strings.HasPrefix(agentIDField, "{") {
		var meta map[string]interface{}
		if json.Unmarshal([]byte(agentIDField), &meta) == nil {
			if role, ok := meta["role"].(string); ok && role != "" {
				agentTitle = role
			}
		}
	}

	// Get image from request or default
	image, _ := req["image"].(string)
	if image == "" {
		if img, ok := s.db.GetSetting("default_agent_image"); ok {
			image = img
		} else {
			image = "claude-agent"
		}
	}

	slog.Info("Processing launch request", "id", reqID, "type", reqType, "image", image)

	// Claim it
	s.db.UpdateLaunchRequest(int(reqID), map[string]interface{}{
		"status":    "claimed",
		"claimed_at": db.Now(),
	})

	var err error
	switch reqType {
	case "new":
		err = s.spawnNewAgent(reqID, folderPath, image, agentTitle, req)
	case "resume":
		err = s.spawnResumeAgent(reqID, resumeAgentID, folderPath, image, agentTitle)
	case "terminate":
		err = s.terminateAgent(req)
	default:
		slog.Warn("Unknown launch request type", "type", reqType, "id", reqID)
		err = fmt.Errorf("unknown type: %s", reqType)
	}

	if err != nil {
		slog.Error("Launch request failed", "id", reqID, "err", err)
		s.db.UpdateLaunchRequest(int(reqID), map[string]interface{}{
			"status":       "failed",
			"completed_at": db.Now(),
		})
	} else {
		s.db.UpdateLaunchRequest(int(reqID), map[string]interface{}{
			"status":       "completed",
			"completed_at": db.Now(),
		})
		slog.Info("Launch request completed", "id", reqID)
	}

	updated := s.db.GetLaunchRequest(int(reqID))
	s.sse.Broadcast("launch-request-updated", updated)
}

func (s *Spawner) spawnNewAgent(reqID int64, folderPath, image, agentTitle string, req map[string]interface{}) error {
	// Read settings from DB
	workspaceRoot, _ := s.db.GetSetting("workspace_root")
	claudeConfigPath, _ := s.db.GetSetting("claude_config_path")
	memLimit, _ := s.db.GetSetting("agent_memory_limit")
	cpuLimit, _ := s.db.GetSetting("agent_cpu_limit")

	if workspaceRoot == "" {
		return fmt.Errorf("workspace_root not configured — complete setup wizard first")
	}
	if claudeConfigPath == "" {
		return fmt.Errorf("claude_config_path not configured — complete setup wizard first")
	}
	if memLimit == "" {
		memLimit = "2g"
	}
	if cpuLimit == "" {
		cpuLimit = "1"
	}

	containerName := fmt.Sprintf("cam-agent-%d", reqID)

	agentURL := "http://host.docker.internal:9222"

	// Extract project context from metadata for agent-to-agent communication
	projectID := ""
	parentAgentID := ""
	if agentIDField, ok := req["agent_id"].(string); ok && strings.HasPrefix(agentIDField, "{") {
		var meta map[string]interface{}
		if json.Unmarshal([]byte(agentIDField), &meta) == nil {
			if pid, ok := meta["project_id"].(string); ok {
				projectID = pid
			}
			if paid, ok := meta["parent_agent_id"].(string); ok {
				parentAgentID = paid
			}
		}
	}

	// Build workspace path: folderPath > project's folder_path > workspace root
	hostPath := workspaceRoot
	if folderPath != "" {
		hostPath = workspaceRoot + "/" + strings.TrimPrefix(folderPath, "/")
	} else if projectID != "" {
		// Sub-agent with no explicit folder — inherit from project
		if project := s.db.GetProject(projectID); project != nil {
			if pf, ok := project["folder_path"].(string); ok && pf != "" {
				hostPath = workspaceRoot + "/" + strings.TrimPrefix(pf, "/")
			}
		}
	}

	// Get dashboard API key so agents can authenticate with all endpoints
	dashboardKey, _ := s.db.GetSetting("api_key")

	// Mount credentials read-only for OAuth auth
	args := []string{
		"run", "-d",
		"--name", containerName,
		"-v", hostPath + ":/workspace",
		"-v", claudeConfigPath + "/.claude/.credentials.json:/home/agent/.claude/.credentials.json:ro",
		"-w", "/workspace",
		"-e", "AGENT_URL=" + agentURL,
		"-e", "AGENT_TITLE=" + agentTitle,
		"-e", "DASHBOARD_API_KEY=" + dashboardKey,
		"--memory", memLimit,
		"--cpus", cpuLimit,
	}

	// Pass project context for agent-to-agent communication
	if projectID != "" {
		args = append(args, "-e", "PROJECT_ID="+projectID)
	}
	if parentAgentID != "" {
		args = append(args, "-e", "PARENT_AGENT_ID="+parentAgentID)
	}

	args = append(args, image)

	return s.runDocker(args, containerName)
}

func (s *Spawner) spawnResumeAgent(reqID int64, agentID, folderPath, image, agentTitle string) error {
	workspaceRoot, _ := s.db.GetSetting("workspace_root")
	claudeConfigPath, _ := s.db.GetSetting("claude_config_path")
	memLimit, _ := s.db.GetSetting("agent_memory_limit")
	cpuLimit, _ := s.db.GetSetting("agent_cpu_limit")

	if workspaceRoot == "" || claudeConfigPath == "" {
		return fmt.Errorf("setup not complete")
	}
	if memLimit == "" {
		memLimit = "2g"
	}
	if cpuLimit == "" {
		cpuLimit = "1"
	}

	// Try to get agent's stored cwd
	if folderPath == "" {
		if agent := s.db.GetAgent(agentID); agent != nil {
			if cwd, ok := agent["cwd"].(string); ok {
				folderPath = cwd
			}
		}
	}

	containerName := fmt.Sprintf("cam-agent-%d", reqID)
	hostPath := workspaceRoot
	if folderPath != "" {
		hostPath = workspaceRoot + "/" + strings.TrimPrefix(folderPath, "/")
	}

	agentURL := "http://host.docker.internal:9222"

	args := []string{
		"run", "-d", "-it",
		"--name", containerName,
		"-v", hostPath + ":/workspace",
		"-v", claudeConfigPath + "/.claude/.credentials.json:/home/agent/.claude/.credentials.json:ro",
		"-v", claudeConfigPath + "/.claude.json:/home/agent/.claude.json",
		"-w", "/workspace",
		"-e", "AGENT_URL=" + agentURL,
		"--memory", memLimit,
		"--cpus", cpuLimit,
		"-e", "POLL_INTERVAL=10",
		image,
	}

	return s.runDocker(args, containerName)
}

func (s *Spawner) terminateAgent(req map[string]interface{}) error {
	// Try container_id from the target agent
	resumeAgentID, _ := req["resume_agent_id"].(string)
	if resumeAgentID == "" {
		return fmt.Errorf("no agent ID for terminate")
	}

	agent := s.db.GetAgent(resumeAgentID)
	if agent == nil {
		return fmt.Errorf("agent not found: %s", resumeAgentID)
	}

	containerID, _ := agent["container_id"].(string)
	if containerID == "" {
		slog.Warn("No container_id for agent, trying name pattern", "agentId", resumeAgentID)
		// Try to find by name pattern (cam-agent-*)
		// Just stop all containers matching the agent
		return nil
	}

	// Stop and remove
	exec.Command("docker", "stop", containerID).Run()
	exec.Command("docker", "rm", "-f", containerID).Run()

	slog.Info("Terminated agent container", "agentId", resumeAgentID, "containerId", containerID)
	return nil
}

func (s *Spawner) runDocker(args []string, containerName string) error {
	slog.Info("Spawning container", "name", containerName, "args", args)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run failed: %w — output: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))
	slog.Info("Container spawned", "name", containerName, "id", containerID[:12])

	return nil
}
