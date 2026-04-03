package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"claude-agent-manager/internal/db"
)

// gzipWriter wraps an http.ResponseWriter to provide gzip compression.
type gzipWriter struct {
	io.Writer
	http.ResponseWriter
}

func (g gzipWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip compression for SSE
		if r.URL.Path == "/api/events" {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		next.ServeHTTP(gzipWriter{Writer: gz, ResponseWriter: w}, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SpawnerNotifier is the interface the spawner exposes to routes.
type SpawnerNotifier interface {
	Notify(requestID int64)
}

type RouterConfig struct {
	DB              *db.DB
	SSE             *SSEBroker
	Auth            *AuthMiddleware
	WebhookDispatch func(string, map[string]interface{})
	OnStatusChange  func(string, string)
	PushNotify      func(string, string, string)
	GetVapidPubKey  func() string
	StartWorkflow   func(string) (bool, string)
	PauseWorkflow   func(string) (bool, string)
	RetentionStatus func() map[string]interface{}
	RunRetention    func() map[string]interface{}
	HostHomeMount   string
	Spawner         SpawnerNotifier
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	agents := NewAgentRoutes(cfg.DB, cfg.SSE, cfg.WebhookDispatch, cfg.OnStatusChange, cfg.PushNotify)
	launch := NewLaunchRoutes(cfg.DB, cfg.SSE, cfg.Spawner)
	projects := NewProjectRoutes(cfg.DB, cfg.SSE, cfg.Spawner)
	workflows := NewWorkflowRoutes(cfg.DB, cfg.StartWorkflow, cfg.PauseWorkflow)
	webhooks := NewWebhookRoutes(cfg.DB)
	push := NewPushRoutes(cfg.DB, cfg.GetVapidPubKey)
	retention := NewRetentionRoutes(cfg.DB, cfg.RetentionStatus, cfg.RunRetention)
	settings := NewSettingsRoutes(cfg.DB)
	workspace := NewWorkspaceRoutes(cfg.DB)
	roleRoutes := NewRoleRoutes()

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/health/db", func(w http.ResponseWriter, r *http.Request) {
		var result string
		cfg.DB.QueryRow("PRAGMA integrity_check").Scan(&result)
		ok := result == "ok"
		status := "ok"
		if !ok {
			status = "error"
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": status, "details": result})
	})

	// Auth
	mux.HandleFunc("GET /api/auth/key", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"apiKey": cfg.Auth.GetAPIKey()})
	})
	mux.HandleFunc("POST /api/auth/rotate", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"apiKey": cfg.Auth.RotateAPIKey()})
	})

	// SSE
	mux.Handle("GET /api/events", cfg.SSE)

	// Agents
	mux.HandleFunc("GET /api/agents", agents.List)
	mux.HandleFunc("GET /api/agents/analytics", agents.Analytics)
	mux.HandleFunc("GET /api/agents/bootstrap", HandleBootstrap(cfg.DB))
	mux.HandleFunc("GET /api/agents/{id}", agents.Get)
	mux.HandleFunc("PATCH /api/agents/{id}", agents.Patch)
	mux.HandleFunc("DELETE /api/agents/{id}", agents.Delete)
	mux.HandleFunc("POST /api/agents/{id}/updates", agents.updateLimiter.Wrap(agents.PostUpdate))
	mux.HandleFunc("GET /api/agents/{id}/updates", agents.GetUpdates)
	mux.HandleFunc("POST /api/agents/{id}/messages", agents.PostMessage)
	mux.HandleFunc("GET /api/agents/{id}/messages", agents.GetMessages)
	mux.HandleFunc("POST /api/agents/{id}/read", agents.MarkRead)
	mux.HandleFunc("POST /api/agents/{id}/close", agents.Close)
	mux.HandleFunc("POST /api/agents/{id}/resume", agents.Resume)
	mux.HandleFunc("POST /api/agents/{id}/relay", agents.Relay)
	mux.HandleFunc("POST /api/agents/{id}/files", agents.fileLimiter.Wrap(agents.UploadFile))
	mux.HandleFunc("GET /api/agents/{id}/files", agents.ListFiles)
	mux.HandleFunc("GET /api/agents/{id}/files/{fileId}", agents.DownloadFile)
	mux.HandleFunc("GET /api/agents/{id}/export/pdf", agents.ExportPDF)

	// Launch requests
	mux.HandleFunc("POST /api/launch-requests", launch.limiter.Wrap(launch.Create))
	mux.HandleFunc("GET /api/launch-requests", launch.List)
	mux.HandleFunc("PATCH /api/launch-requests/{id}", launch.Update)

	// Folders
	mux.HandleFunc("GET /api/folders", HandleFolders(cfg.HostHomeMount))

	// Projects
	mux.HandleFunc("GET /api/projects", projects.List)
	mux.HandleFunc("POST /api/projects", projects.Create)
	mux.HandleFunc("GET /api/projects/{id}", projects.Get)
	mux.HandleFunc("PATCH /api/projects/{id}", projects.Update)
	mux.HandleFunc("GET /api/projects/{id}/agents", projects.GetAgents)
	mux.HandleFunc("GET /api/projects/{id}/updates", projects.GetUpdates)
	mux.HandleFunc("POST /api/projects/{id}/updates", projects.PostUpdate)
	mux.HandleFunc("POST /api/projects/{id}/start", projects.Start)
	mux.HandleFunc("POST /api/projects/{id}/pause", projects.Pause)
	mux.HandleFunc("POST /api/projects/{id}/complete", projects.Complete)
	mux.HandleFunc("DELETE /api/projects/{id}", projects.Delete)
	mux.HandleFunc("GET /api/projects/{id}/files", projects.ListFiles)
	mux.HandleFunc("POST /api/projects/{id}/spawn-agent", projects.SpawnAgent)
	mux.HandleFunc("GET /api/projects/{id}/unified-timeline", HandleUnifiedTimeline(cfg.DB))

	// Workflows
	mux.HandleFunc("GET /api/workflows", workflows.List)
	mux.HandleFunc("POST /api/workflows", workflows.Create)
	mux.HandleFunc("GET /api/workflows/{id}", workflows.Get)
	mux.HandleFunc("POST /api/workflows/{id}/start", workflows.Start)
	mux.HandleFunc("POST /api/workflows/{id}/pause", workflows.Pause)
	mux.HandleFunc("DELETE /api/workflows/{id}", workflows.Delete)

	// Webhooks
	mux.HandleFunc("GET /api/webhooks", webhooks.List)
	mux.HandleFunc("POST /api/webhooks", webhooks.Create)
	mux.HandleFunc("PATCH /api/webhooks/{id}", webhooks.Update)
	mux.HandleFunc("DELETE /api/webhooks/{id}", webhooks.Delete)
	mux.HandleFunc("POST /api/webhooks/{id}/test", webhooks.Test)

	// Push
	mux.HandleFunc("GET /api/push/vapid-public-key", push.VapidPublicKey)
	mux.HandleFunc("POST /api/push/subscribe", push.Subscribe)
	mux.HandleFunc("POST /api/push/unsubscribe", push.Unsubscribe)

	// Retention
	mux.HandleFunc("GET /api/retention/status", retention.Status)
	mux.HandleFunc("POST /api/retention/run", retention.Run)
	mux.HandleFunc("PATCH /api/retention/settings", retention.UpdateSettings)

	// Settings
	mux.HandleFunc("GET /api/settings/setup-status", settings.SetupStatus)
	mux.HandleFunc("GET /api/settings", settings.Get)
	mux.HandleFunc("PATCH /api/settings", settings.Update)

	// Roles (agent role library)
	mux.HandleFunc("GET /api/roles", roleRoutes.List)
	mux.HandleFunc("GET /api/roles/stats", roleRoutes.Stats)
	mux.HandleFunc("GET /api/roles/{id}", roleRoutes.Get)

	// Workspace files
	mux.HandleFunc("GET /api/workspace/files", workspace.ListFiles)
	mux.HandleFunc("GET /api/workspace/file", workspace.ReadFile)

	// Stack middleware: CORS -> Auth -> Gzip -> Mux
	var handler http.Handler = mux
	handler = gzipMiddleware(handler)
	handler = cfg.Auth.Wrap(handler)
	handler = corsMiddleware(handler)

	return handler
}
