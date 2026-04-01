package api

import (
	"net/http"

	"claude-agent-manager/internal/db"
)

type PushRoutes struct {
	db             *db.DB
	getVapidPubKey func() string
}

func NewPushRoutes(d *db.DB, getVapidPubKey func() string) *PushRoutes {
	return &PushRoutes{db: d, getVapidPubKey: getVapidPubKey}
}

func (p *PushRoutes) VapidPublicKey(w http.ResponseWriter, r *http.Request) {
	key := p.getVapidPubKey()
	writeJSON(w, http.StatusOK, map[string]string{"publicKey": key})
}

func (p *PushRoutes) Subscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if err := readJSON(r, &body); err != nil || body.Endpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "endpoint and keys required"})
		return
	}
	p.db.AddPushSubscription(body.Endpoint, body.Keys.P256dh, body.Keys.Auth)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (p *PushRoutes) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := readJSON(r, &body); err != nil || body.Endpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "endpoint required"})
		return
	}
	p.db.RemovePushSubscription(body.Endpoint)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
