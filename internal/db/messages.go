package db

import (
	"database/sql"
	"fmt"
)

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
