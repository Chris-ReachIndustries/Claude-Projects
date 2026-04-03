package db

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
