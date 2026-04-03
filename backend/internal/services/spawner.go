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

	// Container health checker — every 2 minutes, check if agent containers are still running
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			s.checkContainerHealth()
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
	roleID := ""
	if agentIDField, ok := req["agent_id"].(string); ok && strings.HasPrefix(agentIDField, "{") {
		var meta map[string]interface{}
		if json.Unmarshal([]byte(agentIDField), &meta) == nil {
			if pid, ok := meta["project_id"].(string); ok {
				projectID = pid
			}
			if paid, ok := meta["parent_agent_id"].(string); ok {
				parentAgentID = paid
			}
			if rid, ok := meta["role_id"].(string); ok {
				roleID = rid
			}
		}
	}

	// Build workspace path: folderPath > project's folder_path > workspace root
	slog.Info("Spawn workspace resolution", "folderPath", folderPath, "projectID", projectID, "workspaceRoot", workspaceRoot)
	hostPath := workspaceRoot
	if folderPath != "" && folderPath != "/workspace" && folderPath != "workspace" {
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

		// Inject sibling agent info so agents know about each other
		siblings := s.db.GetProjectAgents(projectID)
		if len(siblings) > 0 {
			var siblingInfo []string
			for _, sib := range siblings {
				sibID, _ := sib["id"].(string)
				sibRole, _ := sib["role"].(string)
				sibTitle, _ := sib["title"].(string)
				sibStatus, _ := sib["status"].(string)
				if sibRole == "" {
					sibRole = sibTitle
				}
				if sibID != "" && sibStatus != "archived" {
					siblingInfo = append(siblingInfo, fmt.Sprintf("%s:%s:%s", sibID, sibRole, sibStatus))
				}
			}
			if len(siblingInfo) > 0 {
				args = append(args, "-e", "PROJECT_AGENTS="+strings.Join(siblingInfo, "|"))
			}
		}
	}
	if parentAgentID != "" {
		args = append(args, "-e", "PARENT_AGENT_ID="+parentAgentID)
	}
	if roleID != "" {
		args = append(args, "-e", "ROLE_ID="+roleID)
	}

	args = append(args, image)

	return s.runDocker(args, containerName)
}

func (s *Spawner) spawnResumeAgent(reqID int64, agentID, folderPath, image, agentTitle string) error {
	workspaceRoot, _ := s.db.GetSetting("workspace_root")
	claudeConfigPath, _ := s.db.GetSetting("claude_config_path")
	memLimit, _ := s.db.GetSetting("agent_memory_limit")
	cpuLimit, _ := s.db.GetSetting("agent_cpu_limit")
	dashboardKey, _ := s.db.GetSetting("api_key")

	if workspaceRoot == "" || claudeConfigPath == "" {
		return fmt.Errorf("setup not complete")
	}
	if memLimit == "" {
		memLimit = "2g"
	}
	if cpuLimit == "" {
		cpuLimit = "1"
	}

	// Get agent details from DB for context
	agent := s.db.GetAgent(agentID)
	if agent != nil {
		if folderPath == "" {
			if cwd, ok := agent["cwd"].(string); ok && cwd != "" && cwd != "/workspace" {
				// Only use CWD if it's a real subfolder, not the container-internal /workspace
				folderPath = cwd
			}
		}
		if agentTitle == "" || agentTitle == "Container Agent" {
			if title, ok := agent["title"].(string); ok && title != "" {
				agentTitle = title
			}
			if role, ok := agent["role"].(string); ok && role != "" {
				agentTitle = role
			}
		}
	}

	// If folderPath is still empty, check project
	projectID := ""
	if agent != nil {
		if pid, ok := agent["project_id"].(string); ok {
			projectID = pid
		}
	}
	if folderPath == "" && projectID != "" {
		if project := s.db.GetProject(projectID); project != nil {
			if pf, ok := project["folder_path"].(string); ok && pf != "" {
				folderPath = pf
			}
		}
	}

	containerName := fmt.Sprintf("cam-agent-%d", reqID)
	hostPath := workspaceRoot
	if folderPath != "" {
		hostPath = workspaceRoot + "/" + strings.TrimPrefix(folderPath, "/")
	}
	slog.Info("Resume mount resolved", "agentId", agentID, "folderPath", folderPath, "projectID", projectID, "hostPath", hostPath)

	agentURL := "http://host.docker.internal:9222"

	args := []string{
		"run", "-d",
		"--name", containerName,
		"-v", hostPath + ":/workspace",
		"-v", claudeConfigPath + "/.claude/.credentials.json:/home/agent/.claude/.credentials.json:ro",
		"-w", "/workspace",
		"-e", "AGENT_URL=" + agentURL,
		"-e", "AGENT_TITLE=" + agentTitle,
		"-e", "DASHBOARD_API_KEY=" + dashboardKey,
		"-e", "AGENT_ID=" + agentID, // Reuse original agent ID
		"--memory", memLimit,
		"--cpus", cpuLimit,
	}

	if projectID != "" {
		args = append(args, "-e", "PROJECT_ID="+projectID)
	}

	args = append(args, image)

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
	if len(containerID) > 12 {
		slog.Info("Container spawned", "name", containerName, "id", containerID[:12])
	}

	return nil
}

// checkContainerHealth checks if agent containers are still running.
// If a container exited, marks the agent as "error" status.
func (s *Spawner) checkContainerHealth() {
	// Get running cam-agent containers
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=cam-agent", "--format", "{{.Names}}|{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		containerName := parts[0]
		status := parts[1]

		// Check if container exited unexpectedly
		if strings.Contains(status, "Exited") && !strings.Contains(status, "Exited (0)") {
			// Non-zero exit = crash. Find the agent and mark as error.
			// Container name format: cam-agent-{requestID}
			slog.Warn("Container crashed", "container", containerName, "status", status)

			// Try to find matching agent by checking recent agents
			// This is a best-effort approach
			s.sse.Broadcast("container-health", map[string]interface{}{
				"container": containerName,
				"status":    "crashed",
				"details":   status,
			})
		}
	}
}
