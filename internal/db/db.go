package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps an *sql.DB with helper methods for the agent manager.
type DB struct {
	*sql.DB
}

// PaginatedResult is a generic paginated response.
type PaginatedResult struct {
	Data       []map[string]interface{} `json:"data"`
	NextCursor interface{}              `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

// Now returns the current time in SQLite-compatible format.
func Now() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

// Open creates or opens the SQLite database, runs migrations, and returns a DB.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL and foreign keys
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			return nil, fmt.Errorf("pragma %s: %w", pragma, err)
		}
	}

	d := &DB{sqlDB}
	if err := d.createTables(); err != nil {
		return nil, err
	}
	if err := d.runMigrations(); err != nil {
		return nil, err
	}
	if err := d.createTriggers(); err != nil {
		return nil, err
	}
	d.backfillComputedFields()
	d.migrateFilesToDisk()

	return d, nil
}

func (d *DB) createTables() error {
	_, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT 'Untitled Agent',
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','idle','working','waiting-for-input','completed','archived')),
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			last_update_at TEXT NOT NULL DEFAULT (datetime('now')),
			update_count INTEGER NOT NULL DEFAULT 0,
			metadata TEXT DEFAULT '{}',
			poll_delay_until TEXT,
			workspace TEXT,
			last_read_at TEXT,
			last_activity_at TEXT,
			cwd TEXT,
			pid INTEGER,
			project_id TEXT,
			role TEXT,
			parent_agent_id TEXT,
			pending_message_count INTEGER DEFAULT 0,
			unread_update_count INTEGER DEFAULT 0,
			latest_summary TEXT,
			latest_message TEXT,
			last_message_at TEXT
		);

		CREATE TABLE IF NOT EXISTS updates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			timestamp TEXT NOT NULL DEFAULT (datetime('now')),
			type TEXT NOT NULL DEFAULT 'text' CHECK(type IN ('text','progress','diagram','error','status')),
			content TEXT NOT NULL DEFAULT '{}',
			summary TEXT
		);

		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			delivered_at TEXT,
			acknowledged_at TEXT,
			content TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','delivered','acknowledged','executed')),
			source TEXT DEFAULT 'user',
			source_agent_id TEXT
		);

		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
			filename TEXT NOT NULL,
			mimetype TEXT NOT NULL,
			data BLOB,
			size INTEGER NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			source TEXT NOT NULL DEFAULT 'user',
			description TEXT NOT NULL DEFAULT '',
			file_path TEXT
		);

		CREATE TABLE IF NOT EXISTS launch_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL DEFAULT 'new' CHECK(type IN ('new','resume','terminate')),
			folder_path TEXT NOT NULL DEFAULT '',
			resume_agent_id TEXT,
			target_pid INTEGER,
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','claimed','completed','failed')),
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			claimed_at TEXT,
			completed_at TEXT,
			agent_id TEXT
		);

		CREATE TABLE IF NOT EXISTS push_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			endpoint TEXT NOT NULL UNIQUE,
			keys_p256dh TEXT NOT NULL,
			keys_auth TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS webhooks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL,
			events TEXT NOT NULL DEFAULT '[]',
			active INTEGER DEFAULT 1,
			created_at TEXT DEFAULT (datetime('now')),
			last_triggered_at TEXT,
			failure_count INTEGER DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			steps TEXT NOT NULL DEFAULT '[]',
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending','running','completed','failed','paused')),
			current_step INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now')),
			started_at TEXT,
			completed_at TEXT,
			metadata TEXT DEFAULT '{}'
		);

		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','active','paused','completed','failed')),
			pm_agent_id TEXT,
			folder_path TEXT NOT NULL DEFAULT '',
			max_concurrent INTEGER DEFAULT 4,
			created_at TEXT DEFAULT (datetime('now')),
			started_at TEXT,
			completed_at TEXT,
			metadata TEXT DEFAULT '{}'
		);

		CREATE TABLE IF NOT EXISTS project_updates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			type TEXT NOT NULL DEFAULT 'info',
			content TEXT NOT NULL,
			timestamp TEXT DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_project_updates_project_id ON project_updates(project_id);
		CREATE INDEX IF NOT EXISTS idx_updates_agent_id ON updates(agent_id);
		CREATE INDEX IF NOT EXISTS idx_messages_agent_id ON messages(agent_id);
		CREATE INDEX IF NOT EXISTS idx_files_agent_id ON files(agent_id);
		CREATE INDEX IF NOT EXISTS idx_launch_requests_status ON launch_requests(status);
	`)
	return err
}

func (d *DB) runMigrations() error {
	// Safe column additions — ignore errors if column already exists
	migrations := []string{
		"ALTER TABLE agents ADD COLUMN poll_delay_until TEXT",
		"ALTER TABLE agents ADD COLUMN workspace TEXT",
		"ALTER TABLE agents ADD COLUMN last_read_at TEXT",
		"ALTER TABLE files ADD COLUMN source TEXT NOT NULL DEFAULT 'user'",
		"ALTER TABLE files ADD COLUMN description TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE agents ADD COLUMN last_activity_at TEXT",
		"ALTER TABLE agents ADD COLUMN cwd TEXT",
		"ALTER TABLE agents ADD COLUMN pid INTEGER",
		"ALTER TABLE launch_requests ADD COLUMN target_pid INTEGER",
		"ALTER TABLE agents ADD COLUMN project_id TEXT",
		"ALTER TABLE agents ADD COLUMN role TEXT",
		"ALTER TABLE agents ADD COLUMN parent_agent_id TEXT",
		"ALTER TABLE messages ADD COLUMN source TEXT DEFAULT 'user'",
		"ALTER TABLE messages ADD COLUMN source_agent_id TEXT",
		"ALTER TABLE agents ADD COLUMN pending_message_count INTEGER DEFAULT 0",
		"ALTER TABLE agents ADD COLUMN unread_update_count INTEGER DEFAULT 0",
		"ALTER TABLE agents ADD COLUMN latest_summary TEXT",
		"ALTER TABLE agents ADD COLUMN latest_message TEXT",
		"ALTER TABLE agents ADD COLUMN last_message_at TEXT",
		"ALTER TABLE files ADD COLUMN file_path TEXT",
		// Container spawner columns
		"ALTER TABLE agents ADD COLUMN container_id TEXT",
		"ALTER TABLE launch_requests ADD COLUMN image TEXT DEFAULT 'claude-agent'",
	}
	for _, m := range migrations {
		d.Exec(m) // ignore "duplicate column" errors
	}

	// Backfill last_activity_at
	d.Exec("UPDATE agents SET last_activity_at = last_update_at WHERE last_activity_at IS NULL")

	// Setup defaults
	setupDefaults := map[string]string{
		"setup_complete":       "false",
		"default_agent_image":  "claude-agent",
		"agent_memory_limit":   "2g",
		"agent_cpu_limit":      "1",
		"max_concurrent_agents": "8",
	}
	for k, v := range setupDefaults {
		if _, exists := d.GetSetting(k); !exists {
			d.SetSetting(k, v)
		}
	}

	return nil
}

func (d *DB) createTriggers() error {
	d.Exec(`
		CREATE TRIGGER IF NOT EXISTS after_message_insert AFTER INSERT ON messages
		BEGIN
			UPDATE agents SET
				pending_message_count = pending_message_count + 1,
				latest_message = NEW.content,
				last_message_at = NEW.created_at
			WHERE id = NEW.agent_id;
		END
	`)
	d.Exec(`
		CREATE TRIGGER IF NOT EXISTS after_update_insert AFTER INSERT ON updates
		BEGIN
			UPDATE agents SET
				unread_update_count = unread_update_count + 1,
				latest_summary = COALESCE(NEW.summary, (SELECT latest_summary FROM agents WHERE id = NEW.agent_id))
			WHERE id = NEW.agent_id;
		END
	`)
	return nil
}

func (d *DB) backfillComputedFields() {
	d.Exec(`
		UPDATE agents SET
			pending_message_count = (SELECT COUNT(*) FROM messages WHERE messages.agent_id = agents.id AND messages.status = 'pending'),
			unread_update_count = (SELECT COUNT(*) FROM updates WHERE updates.agent_id = agents.id AND (agents.last_read_at IS NULL OR updates.timestamp > agents.last_read_at)),
			latest_summary = (SELECT summary FROM updates WHERE updates.agent_id = agents.id ORDER BY timestamp DESC LIMIT 1),
			latest_message = (SELECT content FROM messages WHERE messages.agent_id = agents.id ORDER BY created_at DESC LIMIT 1),
			last_message_at = (SELECT MAX(created_at) FROM messages WHERE messages.agent_id = agents.id)
		WHERE EXISTS (SELECT 1 FROM messages WHERE messages.agent_id = agents.id)
		   OR EXISTS (SELECT 1 FROM updates WHERE updates.agent_id = agents.id)
	`)
}

func (d *DB) migrateFilesToDisk() {
	rows, err := d.Query("SELECT id, agent_id, filename, data FROM files WHERE data IS NOT NULL AND length(data) > 0 AND file_path IS NULL")
	if err != nil {
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var agentID, filename string
		var data []byte
		if err := rows.Scan(&id, &agentID, &filename, &data); err != nil {
			continue
		}
		dir := filepath.Join("data", "files", agentID)
		os.MkdirAll(dir, 0755)
		fp := filepath.Join(dir, fmt.Sprintf("%d_%s", id, filename))
		if err := os.WriteFile(fp, data, 0644); err != nil {
			continue
		}
		d.Exec("UPDATE files SET file_path = ?, data = '' WHERE id = ?", fp, id)
		count++
	}
	if count > 0 {
		slog.Info("Migrated file BLOBs to disk", "count", count)
	}
}

// WALCheckpoint runs a passive WAL checkpoint.
func (d *DB) WALCheckpoint() error {
	_, err := d.Exec("PRAGMA wal_checkpoint(PASSIVE)")
	return err
}

// WALCheckpointTruncate runs a truncating WAL checkpoint for shutdown.
func (d *DB) WALCheckpointTruncate() error {
	_, err := d.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// ---- Helpers ----

// scanRows converts sql.Rows into a slice of maps.
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			v := values[i]
			// Convert []byte to string for JSON serialization
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		result = append(result, row)
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	return result, nil
}

// scanRow converts a single sql.Row result into a map.
func (d *DB) scanRowMap(query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// ---- Agent CRUD ----

func (d *DB) GetAllAgents(limit int, cursor string) PaginatedResult {
	var rows *sql.Rows
	var err error
	if cursor != "" {
		rows, err = d.Query(`
			SELECT a.*, p.name as project_name FROM agents a
			LEFT JOIN projects p ON a.project_id = p.id
			WHERE a.last_update_at < ?
			ORDER BY a.last_update_at DESC LIMIT ?`, cursor, limit)
	} else {
		rows, err = d.Query(`
			SELECT a.*, p.name as project_name FROM agents a
			LEFT JOIN projects p ON a.project_id = p.id
			ORDER BY a.last_update_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		return PaginatedResult{Data: []map[string]interface{}{}}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	var nextCursor interface{}
	if len(data) > 0 {
		nextCursor = data[len(data)-1]["last_update_at"]
	}
	return PaginatedResult{Data: data, NextCursor: nextCursor, HasMore: len(data) == limit}
}

func (d *DB) GetAgent(id string) map[string]interface{} {
	row, _ := d.scanRowMap("SELECT * FROM agents WHERE id = ?", id)
	return row
}

func (d *DB) CreateAgent(id, title string) error {
	_, err := d.Exec("INSERT INTO agents (id, title) VALUES (?, ?)", id, title)
	return err
}

func (d *DB) UpdateAgent(id string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	setClauses := []string{}
	values := []interface{}{}
	for k, v := range fields {
		setClauses = append(setClauses, k+" = ?")
		values = append(values, v)
	}
	values = append(values, id)
	_, err := d.Exec("UPDATE agents SET "+strings.Join(setClauses, ", ")+" WHERE id = ?", values...)
	return err
}

func (d *DB) DeleteAgent(id string) error {
	_, err := d.Exec("DELETE FROM agents WHERE id = ?", id)
	return err
}

// ---- Updates ----

func (d *DB) GetUpdates(agentID string, limit int, before int) PaginatedResult {
	var rows *sql.Rows
	var err error
	if before > 0 {
		rows, err = d.Query("SELECT * FROM updates WHERE agent_id = ? AND id < ? ORDER BY id DESC LIMIT ?", agentID, before, limit)
	} else {
		rows, err = d.Query("SELECT * FROM updates WHERE agent_id = ? ORDER BY id DESC LIMIT ?", agentID, limit)
	}
	if err != nil {
		return PaginatedResult{Data: []map[string]interface{}{}}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	var nextCursor interface{}
	if len(data) > 0 {
		nextCursor = data[len(data)-1]["id"]
	}
	return PaginatedResult{Data: data, NextCursor: nextCursor, HasMore: len(data) == limit}
}

func (d *DB) AddUpdate(agentID, updateType, content string, summary *string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sum interface{}
	if summary != nil {
		sum = *summary
	}
	_, err = tx.Exec("INSERT INTO updates (agent_id, type, content, summary) VALUES (?, ?, ?, ?)", agentID, updateType, content, sum)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE agents SET update_count = update_count + 1, last_update_at = datetime('now'), last_activity_at = datetime('now') WHERE id = ?", agentID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ---- Messages ----

func (d *DB) GetPendingMessages(agentID string) ([]map[string]interface{}, error) {
	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT * FROM messages WHERE agent_id = ? AND status = 'pending'", agentID)
	if err != nil {
		return nil, err
	}
	messages, err := scanRows(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}

	tx.Exec("UPDATE messages SET status = 'delivered', delivered_at = datetime('now') WHERE agent_id = ? AND status = 'pending'", agentID)
	tx.Exec("UPDATE agents SET pending_message_count = 0 WHERE id = ?", agentID)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (d *DB) AddMessage(agentID, content, source string, sourceAgentID *string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var srcAgent interface{}
	if sourceAgentID != nil {
		srcAgent = *sourceAgentID
	}
	_, err = tx.Exec("INSERT INTO messages (agent_id, content, source, source_agent_id) VALUES (?, ?, ?, ?)", agentID, content, source, srcAgent)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE agents SET last_activity_at = datetime('now') WHERE id = ?", agentID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) AcknowledgeMessages(agentID string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tx.Exec("UPDATE messages SET status = 'acknowledged', acknowledged_at = datetime('now') WHERE agent_id = ? AND status = 'delivered'", agentID)
	tx.Exec("UPDATE agents SET pending_message_count = 0 WHERE id = ?", agentID)
	return tx.Commit()
}

func (d *DB) GetMessages(agentID string, limit int, before int) PaginatedResult {
	var rows *sql.Rows
	var err error
	if before > 0 {
		rows, err = d.Query("SELECT * FROM messages WHERE agent_id = ? AND id < ? ORDER BY id DESC LIMIT ?", agentID, before, limit)
	} else {
		rows, err = d.Query("SELECT * FROM messages WHERE agent_id = ? ORDER BY id DESC LIMIT ?", agentID, limit)
	}
	if err != nil {
		return PaginatedResult{Data: []map[string]interface{}{}}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	var nextCursor interface{}
	if len(data) > 0 {
		nextCursor = data[len(data)-1]["id"]
	}
	return PaginatedResult{Data: data, NextCursor: nextCursor, HasMore: len(data) == limit}
}

func (d *DB) GetMessagesByStatus(agentID, status string) []map[string]interface{} {
	rows, err := d.Query("SELECT * FROM messages WHERE agent_id = ? AND status = ? ORDER BY created_at ASC", agentID, status)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	return data
}

func (d *DB) TouchAgentHeartbeat(agentID string) {
	d.Exec("UPDATE agents SET last_update_at = datetime('now') WHERE id = ?", agentID)
}

func (d *DB) ArchiveInactiveAgents(inactiveMinutes int) []string {
	tx, err := d.Begin()
	if err != nil {
		return nil
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id FROM agents
		WHERE status IN ('active', 'idle', 'working', 'waiting-for-input')
		  AND last_update_at < datetime('now', ? || ' minutes')
		  AND pending_message_count = 0
		  AND unread_update_count = 0`,
		fmt.Sprintf("-%d", inactiveMinutes))
	if err != nil {
		return nil
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	for _, id := range ids {
		tx.Exec("UPDATE agents SET status = 'archived', last_update_at = datetime('now') WHERE id = ?", id)
	}
	tx.Commit()
	return ids
}

// ---- Files ----

func (d *DB) AddFile(agentID, filename, mimetype, filePath string, size int64, source, description string) (int64, error) {
	result, err := d.Exec("INSERT INTO files (agent_id, filename, mimetype, data, size, source, description, file_path) VALUES (?, ?, ?, '', ?, ?, ?, ?)",
		agentID, filename, mimetype, size, source, description, filePath)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) GetFile(agentID string, fileID int) map[string]interface{} {
	row, _ := d.scanRowMap("SELECT id, agent_id, filename, mimetype, size, source, description, file_path, created_at FROM files WHERE id = ? AND agent_id = ?", fileID, agentID)
	return row
}

func (d *DB) GetFilesMeta(agentID string, limit int, before int) PaginatedResult {
	var rows *sql.Rows
	var err error
	if before > 0 {
		rows, err = d.Query("SELECT id, agent_id, filename, mimetype, size, source, description, file_path, created_at FROM files WHERE agent_id = ? AND id < ? ORDER BY id DESC LIMIT ?", agentID, before, limit)
	} else {
		rows, err = d.Query("SELECT id, agent_id, filename, mimetype, size, source, description, file_path, created_at FROM files WHERE agent_id = ? ORDER BY id DESC LIMIT ?", agentID, limit)
	}
	if err != nil {
		return PaginatedResult{Data: []map[string]interface{}{}}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	var nextCursor interface{}
	if len(data) > 0 {
		nextCursor = data[len(data)-1]["id"]
	}
	return PaginatedResult{Data: data, NextCursor: nextCursor, HasMore: len(data) == limit}
}

func (d *DB) DeleteAgentFiles(agentID string) []string {
	rows, err := d.Query("SELECT file_path FROM files WHERE agent_id = ? AND file_path IS NOT NULL", agentID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var fp string
		rows.Scan(&fp)
		paths = append(paths, fp)
	}
	return paths
}

// ---- Launch Requests ----

func (d *DB) CreateLaunchRequest(reqType, folderPath string, resumeAgentID *string, targetPID *int) map[string]interface{} {
	var raid, tpid interface{}
	if resumeAgentID != nil {
		raid = *resumeAgentID
	}
	if targetPID != nil {
		tpid = *targetPID
	}
	result, err := d.Exec("INSERT INTO launch_requests (type, folder_path, resume_agent_id, target_pid) VALUES (?, ?, ?, ?)",
		reqType, folderPath, raid, tpid)
	if err != nil {
		return nil
	}
	id, _ := result.LastInsertId()
	return map[string]interface{}{
		"id": id, "type": reqType, "folder_path": folderPath,
		"resume_agent_id": raid, "target_pid": tpid, "status": "pending",
	}
}

func (d *DB) GetLaunchRequestsByStatus(status string) []map[string]interface{} {
	rows, err := d.Query("SELECT * FROM launch_requests WHERE status = ? ORDER BY created_at ASC", status)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	return data
}

func (d *DB) UpdateLaunchRequest(id int, fields map[string]interface{}) {
	if len(fields) == 0 {
		return
	}
	setClauses := []string{}
	values := []interface{}{}
	for k, v := range fields {
		setClauses = append(setClauses, k+" = ?")
		values = append(values, v)
	}
	values = append(values, id)
	d.Exec("UPDATE launch_requests SET "+strings.Join(setClauses, ", ")+" WHERE id = ?", values...)
}

func (d *DB) GetLaunchRequest(id int) map[string]interface{} {
	row, _ := d.scanRowMap("SELECT * FROM launch_requests WHERE id = ?", id)
	return row
}

// ---- Settings ----

func (d *DB) GetSetting(key string) (string, bool) {
	var value string
	err := d.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	return value, true
}

func (d *DB) SetSetting(key, value string) {
	d.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
}

// ---- Push Subscriptions ----

func (d *DB) AddPushSubscription(endpoint, p256dh, auth string) {
	d.Exec("INSERT OR REPLACE INTO push_subscriptions (endpoint, keys_p256dh, keys_auth) VALUES (?, ?, ?)", endpoint, p256dh, auth)
}

func (d *DB) RemovePushSubscription(endpoint string) {
	d.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint)
}

type PushSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

func (d *DB) GetAllPushSubscriptions() []PushSubscription {
	rows, err := d.Query("SELECT endpoint, keys_p256dh, keys_auth FROM push_subscriptions")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		rows.Scan(&s.Endpoint, &s.P256dh, &s.Auth)
		subs = append(subs, s)
	}
	return subs
}

// ---- Projects ----

func (d *DB) GetAllProjects() []map[string]interface{} {
	rows, err := d.Query(`
		SELECT p.*,
			(SELECT COUNT(*) FROM agents WHERE project_id = p.id AND status IN ('active','working','idle','waiting-for-input')) as active_agent_count,
			(SELECT COUNT(*) FROM agents WHERE project_id = p.id) as total_agent_count
		FROM projects p ORDER BY p.created_at DESC`)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	return data
}

func (d *DB) GetProject(id string) map[string]interface{} {
	row, _ := d.scanRowMap(`
		SELECT p.*,
			(SELECT COUNT(*) FROM agents WHERE project_id = p.id AND status IN ('active','working','idle','waiting-for-input')) as active_agent_count,
			(SELECT COUNT(*) FROM agents WHERE project_id = p.id) as total_agent_count
		FROM projects p WHERE p.id = ?`, id)
	return row
}

func (d *DB) CreateProject(id, name, description, folderPath string, maxConcurrent int) error {
	_, err := d.Exec("INSERT INTO projects (id, name, description, folder_path, max_concurrent) VALUES (?, ?, ?, ?, ?)",
		id, name, description, folderPath, maxConcurrent)
	return err
}

func (d *DB) UpdateProject(id string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	setClauses := []string{}
	values := []interface{}{}
	for k, v := range fields {
		setClauses = append(setClauses, k+" = ?")
		values = append(values, v)
	}
	values = append(values, id)
	_, err := d.Exec("UPDATE projects SET "+strings.Join(setClauses, ", ")+" WHERE id = ?", values...)
	return err
}

func (d *DB) DeleteProject(id string) error {
	_, err := d.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (d *DB) GetProjectUpdates(projectID string, limit int, before int) PaginatedResult {
	var rows *sql.Rows
	var err error
	if before > 0 {
		rows, err = d.Query("SELECT * FROM project_updates WHERE project_id = ? AND id < ? ORDER BY id DESC LIMIT ?", projectID, before, limit)
	} else {
		rows, err = d.Query("SELECT * FROM project_updates WHERE project_id = ? ORDER BY id DESC LIMIT ?", projectID, limit)
	}
	if err != nil {
		return PaginatedResult{Data: []map[string]interface{}{}}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	var nextCursor interface{}
	if len(data) > 0 {
		nextCursor = data[len(data)-1]["id"]
	}
	return PaginatedResult{Data: data, NextCursor: nextCursor, HasMore: len(data) == limit}
}

func (d *DB) AddProjectUpdate(projectID, updateType, content string) error {
	_, err := d.Exec("INSERT INTO project_updates (project_id, type, content) VALUES (?, ?, ?)", projectID, updateType, content)
	return err
}

func (d *DB) GetProjectAgents(projectID string) []map[string]interface{} {
	rows, err := d.Query("SELECT * FROM agents WHERE project_id = ? ORDER BY created_at DESC", projectID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	return data
}

func (d *DB) GetActiveProjectAgentCount(projectID string) int {
	var count int
	d.QueryRow("SELECT COUNT(*) FROM agents WHERE project_id = ? AND status IN ('active','working','idle','waiting-for-input')", projectID).Scan(&count)
	return count
}

// ---- Analytics ----

func (d *DB) GetAnalytics() map[string]interface{} {
	var totalAgents, activeNow, updatesToday, messagesToday int
	d.QueryRow("SELECT COUNT(*) FROM agents").Scan(&totalAgents)
	d.QueryRow("SELECT COUNT(*) FROM agents WHERE status IN ('active','working','idle','waiting-for-input')").Scan(&activeNow)
	d.QueryRow("SELECT COUNT(*) FROM updates WHERE timestamp > datetime('now', '-24 hours')").Scan(&updatesToday)
	d.QueryRow("SELECT COUNT(*) FROM messages WHERE created_at > datetime('now', '-24 hours')").Scan(&messagesToday)

	rows, _ := d.Query("SELECT status, COUNT(*) as count FROM agents GROUP BY status")
	var statusCounts []map[string]interface{}
	if rows != nil {
		statusCounts, _ = scanRows(rows)
		rows.Close()
	}
	if statusCounts == nil {
		statusCounts = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"totalAgents":  totalAgents,
		"activeNow":    activeNow,
		"updatesToday": updatesToday,
		"messagesToday": messagesToday,
		"statusCounts": statusCounts,
	}
}

// ---- Webhooks (raw queries used by routes) ----

func (d *DB) GetAllWebhooks() []map[string]interface{} {
	rows, err := d.Query("SELECT * FROM webhooks ORDER BY created_at DESC")
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	return data
}

func (d *DB) GetWebhook(id int) map[string]interface{} {
	row, _ := d.scanRowMap("SELECT * FROM webhooks WHERE id = ?", id)
	return row
}

func (d *DB) CreateWebhook(url string, events []string) (map[string]interface{}, error) {
	eventsJSON, _ := json.Marshal(events)
	result, err := d.Exec("INSERT INTO webhooks (url, events) VALUES (?, ?)", url, string(eventsJSON))
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return d.GetWebhook(int(id)), nil
}

func (d *DB) DeleteWebhook(id int) error {
	_, err := d.Exec("DELETE FROM webhooks WHERE id = ?", id)
	return err
}

// ---- Workflows ----

func (d *DB) GetAllWorkflows() []map[string]interface{} {
	rows, err := d.Query("SELECT * FROM workflows ORDER BY created_at DESC")
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()
	data, _ := scanRows(rows)
	return data
}

func (d *DB) GetWorkflow(id string) map[string]interface{} {
	row, _ := d.scanRowMap("SELECT * FROM workflows WHERE id = ?", id)
	return row
}

func (d *DB) DeleteWorkflow(id string) error {
	_, err := d.Exec("DELETE FROM workflows WHERE id = ?", id)
	return err
}

// ---- Retention helpers ----

func (d *DB) GetArchivedAgentsOlderThan(days int) []string {
	cutoff := fmt.Sprintf("-%d days", days)
	rows, err := d.Query("SELECT id FROM agents WHERE status = 'archived' AND last_update_at < datetime('now', ?)", cutoff)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func (d *DB) GetOldUpdates(agentID string, days int) []int {
	cutoff := fmt.Sprintf("-%d days", days)
	rows, err := d.Query(`
		SELECT id FROM updates
		WHERE agent_id = ? AND timestamp < datetime('now', ?)
		AND id NOT IN (SELECT id FROM updates WHERE agent_id = ? ORDER BY id DESC LIMIT 50)`,
		agentID, cutoff, agentID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func (d *DB) GetOldMessages(agentID string, days int) []int {
	cutoff := fmt.Sprintf("-%d days", days)
	rows, err := d.Query(`
		SELECT id FROM messages
		WHERE agent_id = ? AND status = 'acknowledged' AND created_at < datetime('now', ?)
		AND id NOT IN (SELECT id FROM messages WHERE agent_id = ? ORDER BY id DESC LIMIT 20)`,
		agentID, cutoff, agentID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func (d *DB) GetDistinctAgentIDs(table string) []string {
	rows, err := d.Query(fmt.Sprintf("SELECT DISTINCT agent_id FROM %s", table))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func (d *DB) DeleteByIDs(table string, ids []int) {
	if len(ids) == 0 {
		return
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	d.Exec(fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", table, strings.Join(placeholders, ",")), args...)
}

func (d *DB) AgentExists(id string) bool {
	var n int
	d.QueryRow("SELECT 1 FROM agents WHERE id = ?", id).Scan(&n)
	return n == 1
}

// ---- Active webhook rows ----

type WebhookRow struct {
	ID           int
	URL          string
	Events       string
	FailureCount int
}

func (d *DB) GetActiveWebhooks() []WebhookRow {
	rows, err := d.Query("SELECT id, url, events, failure_count FROM webhooks WHERE active = 1")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var hooks []WebhookRow
	for rows.Next() {
		var h WebhookRow
		rows.Scan(&h.ID, &h.URL, &h.Events, &h.FailureCount)
		hooks = append(hooks, h)
	}
	return hooks
}
