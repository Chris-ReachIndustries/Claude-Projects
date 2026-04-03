package db

import (
	"database/sql"
	"fmt"
	"strings"
)

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

func (d *DB) AgentExists(id string) bool {
	var n int
	d.QueryRow("SELECT 1 FROM agents WHERE id = ?", id).Scan(&n)
	return n == 1
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
		"totalAgents":   totalAgents,
		"activeNow":     activeNow,
		"updatesToday":  updatesToday,
		"messagesToday": messagesToday,
		"statusCounts":  statusCounts,
	}
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

func (d *DB) GetDistinctAgentIDs(table string) []string {
	allowed := map[string]bool{"updates": true, "messages": true, "files": true}
	if !allowed[table] {
		return nil
	}
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
	allowed := map[string]bool{"updates": true, "messages": true, "files": true}
	if !allowed[table] {
		return
	}
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
