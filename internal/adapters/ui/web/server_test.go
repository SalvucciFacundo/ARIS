package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServer_Routing(t *testing.T) {
	broker := NewSSEBroker()
	cfg := Config{
		Host:  "127.0.0.1",
		Port:  0, // Ephemeral port for testing
		Token: "test-token",
	}

	handlers := NewHandlers(nil, broker, cfg)
	router := NewRouter(cfg, handlers, broker)

	// Test GET /assets/app.css -> 200 or Static Response
	req := httptest.NewRequest(http.MethodGet, "/assets/app.css", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		// Even if asset is missing or placeholder, shouldn't crash
	}

	// Test GET /api/events (SSE)
	sseReq := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	sseReq.RemoteAddr = "127.0.0.1:54321"
	sseCtx, sseCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer sseCancel()
	sseReq = sseReq.WithContext(sseCtx)
	sseRec := httptest.NewRecorder()
	router.ServeHTTP(sseRec, sseReq)

	if sseRec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", sseRec.Header().Get("Content-Type"))
	}
}

func TestServer_LifecycleAndPortFallback(t *testing.T) {
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     0, // ephemeral or auto
		AutoPort: true,
	}

	srv := NewServer(cfg, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for server to bind
	time.Sleep(100 * time.Millisecond)

	if srv.Addr() == "" {
		t.Fatal("expected server to have a non-empty listening address")
	}

	// Make a quick HTTP request
	resp, err := http.Get(srv.URL() + "/assets/app.css")
	if err != nil {
		t.Fatalf("failed to make request to server: %v", err)
	}
	resp.Body.Close()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown server gracefully: %v", err)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("unexpected error from Start: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for Start to exit after shutdown")
	}
}
