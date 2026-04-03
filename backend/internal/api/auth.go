package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"claude-agent-manager/internal/db"
)

type AuthMiddleware struct {
	db      *db.DB
	enabled bool
}

func NewAuthMiddleware(d *db.DB, enabled bool) *AuthMiddleware {
	return &AuthMiddleware{db: d, enabled: enabled}
}

func (a *AuthMiddleware) GetAPIKey() string {
	key, ok := a.db.GetSetting("api_key")
	if !ok {
		key = generateKey()
		a.db.SetSetting("api_key", key)
	}
	return key
}

func (a *AuthMiddleware) RotateAPIKey() string {
	key := generateKey()
	a.db.SetSetting("api_key", key)
	return key
}

func generateKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Truly public endpoints that never need auth
var exemptPaths = map[string]bool{
	"/api/health":                true, // Health check
	"/api/settings/setup-status": true, // Setup wizard check
	"/api/auth/key":              true, // Auto-fetch key for browser
}

func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for non-API paths (frontend static files)
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Public endpoints
		if exemptPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// During setup, all endpoints are accessible (wizard needs access)
		if complete, _ := a.db.GetSetting("setup_complete"); complete != "true" {
			next.ServeHTTP(w, r)
			return
		}

		// SSE endpoint accepts token as query param (for EventSource which can't set headers)
		if r.URL.Path == "/api/events" {
			token := r.URL.Query().Get("token")
			if token != "" && timingSafeCompare(token, a.GetAPIKey()) {
				next.ServeHTTP(w, r)
				return
			}
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid or missing API key"})
			return
		}

		// All other endpoints: Bearer token in Authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid or missing API key"})
			return
		}

		token := auth[7:]
		if !timingSafeCompare(token, a.GetAPIKey()) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid or missing API key"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func timingSafeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
