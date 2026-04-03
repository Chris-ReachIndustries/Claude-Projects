package db

// ---- Push Subscriptions ----

func (d *DB) AddPushSubscription(endpoint, p256dh, auth string) {
	d.Exec("INSERT OR REPLACE INTO push_subscriptions (endpoint, keys_p256dh, keys_auth) VALUES (?, ?, ?)", endpoint, p256dh, auth)
}

func (d *DB) RemovePushSubscription(endpoint string) {
	d.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint)
}

type PushSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

func (d *DB) GetAllPushSubscriptions() []PushSubscription {
	rows, err := d.Query("SELECT endpoint, keys_p256dh, keys_auth FROM push_subscriptions")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		rows.Scan(&s.Endpoint, &s.P256dh, &s.Auth)
		subs = append(subs, s)
	}
	return subs
}
