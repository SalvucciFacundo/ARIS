package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"aris/internal/adapters/ui/desktop"
	"aris/internal/adapters/ui/web"
)

func TestWebE2E_ServerLifecycleAndSSE(t *testing.T) {
	cfg := web.Config{
		Host:     "127.0.0.1",
		Port:     0, // Ephemeral port
		AutoPort: true,
	}

	server := web.NewServer(cfg, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Wait for server to bind
	var baseURL string
	for i := 0; i < 50; i++ {
		if server.URL() != "" {
			baseURL = server.URL()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if baseURL == "" {
		t.Fatal("web server failed to start within timeout")
	}

	// 1. Verify GET / returns HTML application shell
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("failed to GET /: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Verify GET /assets/app.css returns static CSS
	cssResp, err := http.Get(baseURL + "/assets/app.css")
	if err != nil {
		t.Fatalf("failed to GET /assets/app.css: %v", err)
	}
	if cssResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /assets/app.css, got %d", cssResp.StatusCode)
	}
	cssResp.Body.Close()

	// 3. Connect to SSE /api/events in background
	sseCtx, sseCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer sseCancel()

	sseReq, err := http.NewRequestWithContext(sseCtx, http.MethodGet, baseURL+"/api/events", nil)
	if err != nil {
		t.Fatalf("failed to create SSE request: %v", err)
	}

	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("failed to connect to SSE stream: %v", err)
	}
	defer sseResp.Body.Close()

	eventsReceived := make(chan string, 10)
	go func() {
		scanner := bufio.NewScanner(sseResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:") {
				eventsReceived <- line
			}
		}
	}()

	// 4. Submit a generation request to /api/generate
	genPayload, _ := json.Marshal(map[string]any{
		"prompt":       "cyberpunk neon cityscape in rain",
		"aspect_ratio": "16:9",
		"backend":      "pollinations",
	})
	postResp, err := http.Post(baseURL+"/api/generate", "application/json", bytes.NewReader(genPayload))
	if err != nil {
		t.Fatalf("failed to POST /api/generate: %v", err)
	}
	if postResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted from /api/generate, got %d", postResp.StatusCode)
	}
	postResp.Body.Close()

	// 5. Verify SSE event arrived
	select {
	case evt := <-eventsReceived:
		if evt == "" {
			t.Fatal("received empty SSE line")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for SSE progress event")
	}

	// 6. Close SSE connection and shutdown server
	sseCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("failed to shutdown server: %v", err)
	}
}

func TestWebE2E_DesktopRunnerIntegration(t *testing.T) {
	runner := desktop.NewRunner(desktop.Config{
		WebCfg: web.Config{
			Host:     "127.0.0.1",
			Port:     0,
			AutoPort: true,
		},
		NoBrowser: true,
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	err := runner.Run(ctx)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected runner error: %v", err)
	}
}
