package db

import "database/sql"

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
