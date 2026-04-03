package api

import (
	"fmt"
	"net/http"

	"claude-agent-manager/internal/db"
)

type SettingsRoutes struct {
	db *db.DB
}

func NewSettingsRoutes(d *db.DB) *SettingsRoutes {
	return &SettingsRoutes{db: d}
}

// SetupStatus returns whether initial setup is complete. Auth-exempt.
func (s *SettingsRoutes) SetupStatus(w http.ResponseWriter, r *http.Request) {
	complete, _ := s.db.GetSetting("setup_complete")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"complete": complete == "true",
	})
}

// Get returns all settings (masks sensitive values).
func (s *SettingsRoutes) Get(w http.ResponseWriter, r *http.Request) {
	keys := []string{
		"setup_complete", "workspace_root", "claude_config_path",
		"default_agent_image", "agent_memory_limit", "agent_cpu_limit",
		"max_concurrent_agents",
	}

	settings := map[string]interface{}{}
	for _, k := range keys {
		if v, ok := s.db.GetSetting(k); ok {
			settings[k] = v
		} else {
			settings[k] = nil
		}
	}

	writeJSON(w, http.StatusOK, settings)
}

// Update patches settings.
func (s *SettingsRoutes) Update(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	allowedKeys := map[string]bool{
		"claude_config_path":    true,
		"workspace_root":        true,
		"default_agent_image":   true,
		"agent_memory_limit":    true,
		"agent_cpu_limit":       true,
		"max_concurrent_agents": true,
		"setup_complete":        true,
	}

	for k, v := range body {
		if !allowedKeys[k] {
			continue
		}
		var strVal string
		switch tv := v.(type) {
		case string:
			strVal = tv
		case float64:
			if tv == float64(int(tv)) {
				strVal = fmt.Sprintf("%d", int(tv))
			} else {
				strVal = fmt.Sprintf("%g", tv)
			}
		case bool:
			strVal = fmt.Sprintf("%t", tv)
		default:
			continue
		}
		s.db.SetSetting(k, strVal)
	}

	s.Get(w, r)
}
