package services

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"

	"claude-agent-manager/internal/db"
)

type WebhookDispatcher struct {
	db *db.DB
}

func NewWebhookDispatcher(d *db.DB) *WebhookDispatcher {
	return &WebhookDispatcher{db: d}
}

func (wd *WebhookDispatcher) Dispatch(event string, data map[string]interface{}) {
	go wd.dispatchAll(event, data)
}

func (wd *WebhookDispatcher) dispatchAll(event string, data map[string]interface{}) {
	hooks := wd.db.GetActiveWebhooks()

	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"agent":     data["agent"],
		"details":   data["details"],
	}

	for _, hook := range hooks {
		var events []string
		json.Unmarshal([]byte(hook.Events), &events)

		found := false
		for _, e := range events {
			if e == event {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		go wd.deliverWithRetry(hook.ID, hook.URL, payload, 0)
	}
}

func (wd *WebhookDispatcher) deliverWithRetry(webhookID int, url string, payload map[string]interface{}, attempt int) {
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))

	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		resp.Body.Close()
		wd.db.Exec("UPDATE webhooks SET failure_count = 0, last_triggered_at = datetime('now') WHERE id = ?", webhookID)
		slog.Info("Webhook delivered", "webhookId", webhookID, "event", payload["event"])
		return
	}
	if resp != nil {
		resp.Body.Close()
	}

	if attempt < 2 { // max 3 attempts (0, 1, 2)
		delay := time.Duration(math.Pow(5, float64(attempt))) * time.Second // 1s, 5s, 25s
		time.Sleep(delay)
		wd.deliverWithRetry(webhookID, url, payload, attempt+1)
		return
	}

	// All retries exhausted
	wd.db.Exec("UPDATE webhooks SET failure_count = failure_count + 1 WHERE id = ?", webhookID)

	// Auto-disable after 10 failures
	var failureCount int
	wd.db.QueryRow("SELECT failure_count FROM webhooks WHERE id = ?", webhookID).Scan(&failureCount)
	if failureCount >= 10 {
		wd.db.Exec("UPDATE webhooks SET active = 0 WHERE id = ?", webhookID)
		slog.Warn("Webhook auto-disabled after consecutive failures", "webhookId", webhookID)
	}

	slog.Error("Webhook delivery failed", "webhookId", webhookID, "attempt", attempt, "err", err)
}
