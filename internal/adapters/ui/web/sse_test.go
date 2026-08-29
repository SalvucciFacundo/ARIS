package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSSEBroker_RegisterUnregisterBroadcast(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go broker.Start(ctx)

	clientCh := broker.Subscribe()
	if clientCh == nil {
		t.Fatal("expected non-nil channel from Subscribe")
	}

	event := SSEEvent{
		Event: "progress",
		Data:  map[string]any{"job_id": "test-1", "percent": 50},
	}

	broker.Broadcast(event)

	select {
	case received := <-clientCh:
		if received.Event != "progress" {
			t.Fatalf("expected event 'progress', got %s", received.Event)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}

	broker.Unsubscribe(clientCh)

	// After unregistering, sending event should not block or panic
	broker.Broadcast(SSEEvent{Event: "test", Data: "hello"})

	select {
	case _, ok := <-clientCh:
		if ok {
			// May have drained previous or closed
		}
	default:
		// channel is either closed or empty
	}
}

func TestSSEBroker_ConcurrentBroadcasting(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go broker.Start(ctx)

	numClients := 10
	var clients []chan SSEEvent
	for i := 0; i < numClients; i++ {
		clients = append(clients, broker.Subscribe())
	}

	var wg sync.WaitGroup
	// 5 concurrent broadcasters
	for b := 0; b < 5; b++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for e := 0; e < 20; e++ {
				broker.Broadcast(SSEEvent{
					Event: "progress",
					Data:  map[string]any{"sender": id, "step": e},
				})
			}
		}(b)
	}

	// Concurrently subscribe and unsubscribe new clients
	for c := 0; c < 5; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := broker.Subscribe()
			time.Sleep(10 * time.Millisecond)
			broker.Unsubscribe(ch)
		}()
	}

	wg.Wait()

	for _, ch := range clients {
		broker.Unsubscribe(ch)
	}
}

func TestSSEBroker_ServeHTTP(t *testing.T) {
	broker := NewSSEBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go broker.Start(ctx)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	reqCtx, reqCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer reqCancel()
	req = req.WithContext(reqCtx)

	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		broker.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	broker.Broadcast(SSEEvent{
		Event: "progress",
		Data:  map[string]any{"job_id": "job-123", "percent": 100},
	})

	<-done

	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("expected Cache-Control no-cache, got %s", rec.Header().Get("Cache-Control"))
	}
}
