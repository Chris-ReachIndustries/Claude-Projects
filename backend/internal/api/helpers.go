package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func parseIntQuery(r *http.Request, key string, defaultVal, max int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultVal
	}
	if n > max {
		return max
	}
	return n
}

func pathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}
