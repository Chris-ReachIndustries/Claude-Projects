package db

// ---- Settings ----

func (d *DB) GetSetting(key string) (string, bool) {
	var value string
	err := d.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", false
	}
	return value, true
}

func (d *DB) SetSetting(key, value string) {
	d.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
}

