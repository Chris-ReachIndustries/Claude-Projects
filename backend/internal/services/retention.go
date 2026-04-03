package services

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"claude-agent-manager/internal/db"
)

type RetentionService struct {
	db           *db.DB
	mu           sync.Mutex
	lastRunAt    *string
	lastRunStats map[string]interface{}
}

func NewRetentionService(d *db.DB) *RetentionService {
	rs := &RetentionService{db: d}
	rs.ensureDefaults()
	return rs
}

func (rs *RetentionService) ensureDefaults() {
	defaults := map[string]interface{}{
		"retention_archive_days": 30,
		"retention_update_days":  30,
		"retention_message_days": 30,
		"retention_enabled":      true,
		"retention_dry_run":      true,
	}
	for k, v := range defaults {
		if _, ok := rs.db.GetSetting(k); !ok {
			b, _ := json.Marshal(v)
			rs.db.SetSetting(k, string(b))
		}
	}
}

func (rs *RetentionService) GetSettings() map[string]interface{} {
	settings := map[string]interface{}{}
	keys := []string{"retention_archive_days", "retention_update_days", "retention_message_days", "retention_enabled", "retention_dry_run"}
	defaults := map[string]interface{}{
		"retention_archive_days": 30.0,
		"retention_update_days":  30.0,
		"retention_message_days": 30.0,
		"retention_enabled":      true,
		"retention_dry_run":      true,
	}
	for _, k := range keys {
		raw, ok := rs.db.GetSetting(k)
		if ok {
			var v interface{}
			json.Unmarshal([]byte(raw), &v)
			settings[k] = v
		} else {
			settings[k] = defaults[k]
		}
	}
	return settings
}

func (rs *RetentionService) GetStatus() map[string]interface{} {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return map[string]interface{}{
		"settings":     rs.GetSettings(),
		"lastRunAt":    rs.lastRunAt,
		"lastRunStats": rs.lastRunStats,
	}
}

func (rs *RetentionService) Run() map[string]interface{} {
	settings := rs.GetSettings()
	stats := map[string]interface{}{
		"agentsDeleted":   0,
		"updatesDeleted":  0,
		"messagesDeleted": 0,
		"filesDeleted":    0,
	}

	enabled, _ := settings["retention_enabled"].(bool)
	if !enabled {
		slog.Info("Retention is disabled, skipping")
		return stats
	}
	dryRun, _ := settings["retention_dry_run"].(bool)
	prefix := ""
	if dryRun {
		prefix = "[DRY RUN] "
	}

	archiveDays := intFromFloat(settings["retention_archive_days"], 30)
	updateDays := intFromFloat(settings["retention_update_days"], 30)
	messageDays := intFromFloat(settings["retention_message_days"], 30)

	agentsDeleted := 0
	updatesDeleted := 0
	messagesDeleted := 0
	filesDeleted := 0

	// 1. Delete old archived agents
	oldAgents := rs.db.GetArchivedAgentsOlderThan(archiveDays)
	for _, agentID := range oldAgents {
		filePaths := rs.db.DeleteAgentFiles(agentID)
		for _, fp := range filePaths {
			slog.Info(prefix+"Retention: deleting file", "path", fp)
			if !dryRun {
				os.Remove(fp)
			}
			filesDeleted++
		}
		filesDir := filepath.Join("data", "files", agentID)
		if !dryRun {
			os.RemoveAll(filesDir)
			rs.db.DeleteAgent(agentID)
		}
		agentsDeleted++
	}

	// 2. Delete old updates
	for _, agentID := range rs.db.GetDistinctAgentIDs("updates") {
		ids := rs.db.GetOldUpdates(agentID, updateDays)
		if len(ids) > 0 {
			slog.Info(prefix+"Retention: deleting old updates", "agent", agentID, "count", len(ids))
			if !dryRun {
				rs.db.DeleteByIDs("updates", ids)
			}
			updatesDeleted += len(ids)
		}
	}

	// 3. Delete old acknowledged messages
	for _, agentID := range rs.db.GetDistinctAgentIDs("messages") {
		ids := rs.db.GetOldMessages(agentID, messageDays)
		if len(ids) > 0 {
			slog.Info(prefix+"Retention: deleting old messages", "agent", agentID, "count", len(ids))
			if !dryRun {
				rs.db.DeleteByIDs("messages", ids)
			}
			messagesDeleted += len(ids)
		}
	}

	// 4. Clean orphaned file directories
	filesBaseDir := filepath.Join("data", "files")
	if entries, err := os.ReadDir(filesBaseDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !rs.db.AgentExists(e.Name()) {
				orphanDir := filepath.Join(filesBaseDir, e.Name())
				orphanFiles, _ := os.ReadDir(orphanDir)
				slog.Info(prefix+"Retention: cleaning orphaned files", "agent", e.Name(), "count", len(orphanFiles))
				if !dryRun {
					os.RemoveAll(orphanDir)
				}
				filesDeleted += len(orphanFiles)
			}
		}
	}

	stats["agentsDeleted"] = agentsDeleted
	stats["updatesDeleted"] = updatesDeleted
	stats["messagesDeleted"] = messagesDeleted
	stats["filesDeleted"] = filesDeleted

	rs.mu.Lock()
	now := time.Now().UTC().Format(time.RFC3339)
	rs.lastRunAt = &now
	rs.lastRunStats = stats
	rs.mu.Unlock()

	slog.Info(prefix+"Retention run complete", "stats", stats)
	return stats
}

func (rs *RetentionService) Start() {
	// Run after short delay
	go func() {
		time.Sleep(5 * time.Second)
		rs.Run()
	}()

	// Run daily
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			rs.Run()
		}
	}()
}

func intFromFloat(v interface{}, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return def
	}
}
