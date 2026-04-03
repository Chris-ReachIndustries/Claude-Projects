package api

import (
	"encoding/json"
	"net/http"

	"claude-agent-manager/internal/db"
)

type RetentionRoutes struct {
	db           *db.DB
	getStatus    func() map[string]interface{}
	runRetention func() map[string]interface{}
}

func NewRetentionRoutes(d *db.DB, getStatus func() map[string]interface{}, runRetention func() map[string]interface{}) *RetentionRoutes {
	return &RetentionRoutes{db: d, getStatus: getStatus, runRetention: runRetention}
}

func (rt *RetentionRoutes) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, rt.getStatus())
}

func (rt *RetentionRoutes) Run(w http.ResponseWriter, r *http.Request) {
	stats := rt.runRetention()
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "stats": stats})
}

func (rt *RetentionRoutes) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	validKeys := []string{
		"retention_archive_days", "retention_update_days", "retention_message_days",
		"retention_enabled", "retention_dry_run",
	}
	for _, key := range validKeys {
		if v, ok := body[key]; ok {
			b, _ := json.Marshal(v)
			rt.db.SetSetting(key, string(b))
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "settings": rt.getStatus()})
}
