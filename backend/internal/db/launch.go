package db

import "strings"

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
