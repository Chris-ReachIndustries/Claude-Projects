package db

import (
	"database/sql"
	"fmt"
)

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
