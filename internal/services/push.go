package services

import (
	"log/slog"

	"claude-agent-manager/internal/db"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type PushService struct {
	db *db.DB
}

func NewPushService(d *db.DB) *PushService {
	ps := &PushService{db: d}
	ps.init()
	return ps
}

func (ps *PushService) init() {
	_, hasPub := ps.db.GetSetting("vapid_public_key")
	_, hasPriv := ps.db.GetSetting("vapid_private_key")

	if !hasPub || !hasPriv {
		priv, pub, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			slog.Error("Failed to generate VAPID keys", "err", err)
			return
		}
		ps.db.SetSetting("vapid_public_key", pub)
		ps.db.SetSetting("vapid_private_key", priv)
		slog.Info("Generated new VAPID keys")
	}
	slog.Info("Web Push initialized")
}

func (ps *PushService) GetVapidPublicKey() string {
	key, _ := ps.db.GetSetting("vapid_public_key")
	return key
}

func (ps *PushService) SendToAll(title, body, url string) {
	subs := ps.db.GetAllPushSubscriptions()
	if len(subs) == 0 {
		return
	}

	pubKey, _ := ps.db.GetSetting("vapid_public_key")
	privKey, _ := ps.db.GetSetting("vapid_private_key")
	if pubKey == "" || privKey == "" {
		return
	}

	payload := `{"title":"` + title + `","body":"` + body + `","url":"` + url + `"}`

	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification([]byte(payload), s, &webpush.Options{
			Subscriber:      "mailto:agent-manager@localhost",
			VAPIDPublicKey:  pubKey,
			VAPIDPrivateKey: privKey,
		})
		if err != nil {
			slog.Error("Push notification failed", "err", err, "endpoint", sub.Endpoint)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			ps.db.RemovePushSubscription(sub.Endpoint)
			slog.Info("Removed expired push subscription")
		}
	}
}
