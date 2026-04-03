package db

import "encoding/json"

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
