package api

import (
	"net/http"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count    int
	windowStart time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateLimitEntry
	windowMs int64
	max      int
	keyFunc  func(*http.Request) string
}

func NewRateLimiter(windowMs int64, max int, keyFunc func(*http.Request) string) *RateLimiter {
	rl := &RateLimiter{
		entries:  make(map[string]*rateLimitEntry),
		windowMs: windowMs,
		max:      max,
		keyFunc:  keyFunc,
	}
	// Cleanup expired entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	window := time.Duration(rl.windowMs) * time.Millisecond
	for k, e := range rl.entries {
		if now.Sub(e.windowStart) > window {
			delete(rl.entries, k)
		}
	}
}

func (rl *RateLimiter) Allow(r *http.Request) bool {
	key := rl.keyFunc(r)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window := time.Duration(rl.windowMs) * time.Millisecond

	e, ok := rl.entries[key]
	if !ok || now.Sub(e.windowStart) > window {
		rl.entries[key] = &rateLimitEntry{count: 1, windowStart: now}
		return true
	}

	e.count++
	return e.count <= rl.max
}

func (rl *RateLimiter) Wrap(next http.HandlerFunc) http.HandlerFunc {
	retryAfter := rl.windowMs / 1000
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"error":      "Rate limit exceeded",
				"retryAfter": retryAfter,
			})
			return
		}
		next(w, r)
	}
}
