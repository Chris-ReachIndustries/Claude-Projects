package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"claude-agent-manager/internal/db"
)

type WebhookRoutes struct {
	db *db.DB
}

func NewWebhookRoutes(d *db.DB) *WebhookRoutes {
	return &WebhookRoutes{db: d}
}

func (wh *WebhookRoutes) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, wh.db.GetAllWebhooks())
}

func (wh *WebhookRoutes) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := readJSON(r, &body); err != nil || body.URL == "" || len(body.Events) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url and events required"})
		return
	}
	webhook, err := wh.db.CreateWebhook(body.URL, body.Events)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create webhook"})
		return
	}
	writeJSON(w, http.StatusCreated, webhook)
}

func (wh *WebhookRoutes) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(pathParam(r, "id"))
	existing := wh.db.GetWebhook(id)
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Webhook not found"})
		return
	}

	var body map[string]interface{}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	setClauses := map[string]interface{}{}
	if url, ok := body["url"].(string); ok {
		setClauses["url"] = url
	}
	if events, ok := body["events"].([]interface{}); ok {
		b, _ := json.Marshal(events)
		setClauses["events"] = string(b)
	}
	if active, ok := body["active"].(bool); ok {
		if active {
			setClauses["active"] = 1
			setClauses["failure_count"] = 0
		} else {
			setClauses["active"] = 0
		}
	}

	if len(setClauses) > 0 {
		wh.db.UpdateLaunchRequest(id, setClauses) // reuse generic update
		// Actually need webhook-specific update
		for k, v := range setClauses {
			wh.db.Exec("UPDATE webhooks SET "+k+" = ? WHERE id = ?", v, id)
		}
	}

	writeJSON(w, http.StatusOK, wh.db.GetWebhook(id))
}

func (wh *WebhookRoutes) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(pathParam(r, "id"))
	if wh.db.GetWebhook(id) == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Webhook not found"})
		return
	}
	wh.db.DeleteWebhook(id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (wh *WebhookRoutes) Test(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(pathParam(r, "id"))
	webhook := wh.db.GetWebhook(id)
	if webhook == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Webhook not found"})
		return
	}

	url, _ := webhook["url"].(string)
	testPayload := map[string]interface{}{
		"event":     "webhook.test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"agent":     map[string]string{"id": "test-agent", "title": "Test Agent", "status": "active"},
		"details":   map[string]string{"message": "This is a test webhook delivery"},
	}

	body, _ := json.Marshal(testPayload)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "Failed to test webhook", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": resp.StatusCode >= 200 && resp.StatusCode < 300, "status": resp.StatusCode, "statusText": resp.Status,
	})
}
