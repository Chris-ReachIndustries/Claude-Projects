package db

import (
	"database/sql"
	"strings"
)

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
