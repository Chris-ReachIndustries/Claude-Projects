package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

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
			type TEXT NOT NULL DEFAULT 'text' CHECK(type IN ('text','progress','diagram','error','status','tool','thinking','info','message')),
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
		CREATE INDEX IF NOT EXISTS idx_agents_project ON agents(project_id);
		CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
		CREATE INDEX IF NOT EXISTS idx_messages_agent_status ON messages(agent_id, status);
		CREATE INDEX IF NOT EXISTS idx_updates_agent ON updates(agent_id);
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

	// New feature columns
	d.Exec("ALTER TABLE projects ADD COLUMN autonomy_mode TEXT DEFAULT 'autonomous'")
	d.Exec("ALTER TABLE agents ADD COLUMN tokens_in INTEGER DEFAULT 0")
	d.Exec("ALTER TABLE agents ADD COLUMN tokens_out INTEGER DEFAULT 0")
	d.Exec("ALTER TABLE agents ADD COLUMN model TEXT")
	d.Exec("ALTER TABLE launch_requests ADD COLUMN model TEXT")

	// Remove CHECK constraint on updates.type to allow 'tool', 'thinking', 'info', 'message'
	// SQLite doesn't support ALTER CHECK, so recreate the table
	var hasCheck int
	d.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='updates' AND sql LIKE '%CHECK%type%'").Scan(&hasCheck)
	if hasCheck > 0 {
		d.Exec("ALTER TABLE updates RENAME TO updates_old")
		d.Exec(`CREATE TABLE IF NOT EXISTS updates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'text',
			content TEXT NOT NULL DEFAULT '',
			summary TEXT,
			timestamp DATETIME DEFAULT (datetime('now')),
			FOREIGN KEY (agent_id) REFERENCES agents(id)
		)`)
		d.Exec("INSERT INTO updates SELECT * FROM updates_old")
		d.Exec("DROP TABLE updates_old")
		d.Exec("CREATE INDEX IF NOT EXISTS idx_updates_agent ON updates(agent_id)")
	}

	// Backfill last_activity_at
	d.Exec("UPDATE agents SET last_activity_at = last_update_at WHERE last_activity_at IS NULL")

	// Setup defaults
	setupDefaults := map[string]string{
		"setup_complete":        "false",
		"default_agent_image":   "claude-agent",
		"agent_memory_limit":    "2g",
		"agent_cpu_limit":       "1",
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
