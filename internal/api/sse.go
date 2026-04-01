package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const maxSSEClients = 10

type sseClient struct {
	w       http.ResponseWriter
	flusher http.Flusher
	done    chan struct{}
}

type SSEBroker struct {
	mu      sync.Mutex
	clients map[*sseClient]struct{}
	stop    chan struct{}
}

func NewSSEBroker() *SSEBroker {
	b := &SSEBroker{
		clients: make(map[*sseClient]struct{}),
		stop:    make(chan struct{}),
	}
	go b.keepAlive()
	return b
}

func (b *SSEBroker) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.mu.Lock()
			for c := range b.clients {
				select {
				case <-c.done:
					delete(b.clients, c)
				default:
					fmt.Fprint(c.w, ":ping\n\n")
					c.flusher.Flush()
				}
			}
			b.mu.Unlock()
		case <-b.stop:
			return
		}
	}
}

func (b *SSEBroker) ClientCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.clients)
}

func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if b.ClientCount() >= maxSSEClients {
		http.Error(w, `{"error":"Too many SSE connections"}`, http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprint(w, ":ok\n\n")
	flusher.Flush()

	client := &sseClient{w: w, flusher: flusher, done: make(chan struct{})}
	b.mu.Lock()
	b.clients[client] = struct{}{}
	b.mu.Unlock()

	// Block until client disconnects
	<-r.Context().Done()

	close(client.done)
	b.mu.Lock()
	delete(b.clients, client)
	b.mu.Unlock()
}

func (b *SSEBroker) Broadcast(event string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("SSE marshal error", "err", err)
		return
	}
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event, payload)

	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.clients {
		select {
		case <-c.done:
			delete(b.clients, c)
		default:
			fmt.Fprint(c.w, msg)
			c.flusher.Flush()
		}
	}
}

func (b *SSEBroker) Close() {
	close(b.stop)
}
