package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEEvent represents a single Server-Sent Event payload.
type SSEEvent struct {
	Event string `json:"event,omitempty"`
	Data  any    `json:"data"`
	ID    string `json:"id,omitempty"`
	Retry int    `json:"retry,omitempty"`
}

// SSEBroker manages client subscriptions and event fan-out.
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]struct{}
	closed  bool
}

// NewSSEBroker constructs a new thread-safe SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan SSEEvent]struct{}),
	}
}

// Start begins the 15-second heartbeat loop until ctx is canceled.
func (b *SSEBroker) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.closeAll()
			return
		case <-ticker.C:
			b.Broadcast(SSEEvent{
				Event: "ping",
				Data:  ":ping",
			})
		}
	}
}

// Subscribe registers a new event channel and returns it.
func (b *SSEBroker) Subscribe() chan SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan SSEEvent, 64)
	if !b.closed {
		b.clients[ch] = struct{}{}
	} else {
		close(ch)
	}
	return ch
}

// Unsubscribe safely deregisters and drains the client channel.
func (b *SSEBroker) Unsubscribe(ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.clients[ch]; ok {
		delete(b.clients, ch)
		close(ch)
	}
}

// Broadcast sends an event to all subscribed clients without blocking.
func (b *SSEBroker) Broadcast(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	for ch := range b.clients {
		select {
		case ch <- event:
		default:
			// Client buffer is full; drop event to avoid blocking other subscribers
		}
	}
}

func (b *SSEBroker) closeAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true
	for ch := range b.clients {
		close(ch)
	}
	b.clients = make(map[chan SSEEvent]struct{})
}

// ServeHTTP handles incoming HTTP clients connecting to /api/events.
func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	clientCh := b.Subscribe()
	defer b.Unsubscribe(clientCh)

	ctx := r.Context()

	// Initial comment to confirm connection
	_, _ = fmt.Fprintf(w, ":connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-clientCh:
			if !ok {
				return
			}

			if event.ID != "" {
				_, _ = fmt.Fprintf(w, "id: %s\n", event.ID)
			}
			if event.Event != "" {
				_, _ = fmt.Fprintf(w, "event: %s\n", event.Event)
			}
			if event.Retry > 0 {
				_, _ = fmt.Fprintf(w, "retry: %d\n", event.Retry)
			}

			var payload []byte
			switch v := event.Data.(type) {
			case string:
				payload = []byte(v)
			case []byte:
				payload = v
			default:
				payload, _ = json.Marshal(v)
			}

			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(payload))
			flusher.Flush()
		}
	}
}
