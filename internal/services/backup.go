package services

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"claude-agent-manager/internal/db"
)

func StartBackupScheduler(d *db.DB) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/agents.db"
	}
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	os.MkdirAll(backupDir, 0755)

	intervalHours := 12
	if v := os.Getenv("BACKUP_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			intervalHours = n
		}
	}
	retentionDays := 7
	if v := os.Getenv("BACKUP_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retentionDays = n
		}
	}

	runBackup := func() {
		timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
		backupPath := filepath.Join(backupDir, "agents_"+timestamp+".db")

		// Use SQLite backup via VACUUM INTO
		_, err := d.Exec("VACUUM INTO ?", backupPath)
		if err != nil {
			slog.Error("Database backup failed", "err", err)
			return
		}
		slog.Info("Database backup completed", "path", backupPath)

		// Clean old backups
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		entries, _ := os.ReadDir(backupDir)
		for _, e := range entries {
			if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(backupDir, e.Name()))
				slog.Info("Deleted old backup", "file", e.Name())
			}
		}
	}

	// Run immediately
	go runBackup()

	// Schedule periodic backups
	go func() {
		ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
		for range ticker.C {
			runBackup()
		}
	}()

	// WAL checkpoint every hour
	go func() {
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			if err := d.WALCheckpoint(); err != nil {
				slog.Error("WAL checkpoint failed", "err", err)
			} else {
				slog.Info("WAL checkpoint completed")
			}
		}
	}()
}
