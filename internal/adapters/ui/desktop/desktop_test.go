package desktop

import (
	"context"
	"testing"
	"time"

	"aris/internal/adapters/ui/web"
)

func TestDesktop_FallbackBrowserURLConstruction(t *testing.T) {
	url := BuildLaunchURL("http://127.0.0.1:8080", "secret-token")
	expected := "http://127.0.0.1:8080?token=secret-token"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}

	urlNoToken := BuildLaunchURL("http://127.0.0.1:8080", "")
	if urlNoToken != "http://127.0.0.1:8080" {
		t.Fatalf("expected http://127.0.0.1:8080, got %s", urlNoToken)
	}

	urlExistingQuery := BuildLaunchURL("https://vps.aris.ai:8443/app?theme=dark", "token123")
	if urlExistingQuery != "https://vps.aris.ai:8443/app?theme=dark&token=token123" {
		t.Fatalf("expected https://vps.aris.ai:8443/app?theme=dark&token=token123, got %s", urlExistingQuery)
	}
}

func TestDesktop_RunnerRemoteMode(t *testing.T) {
	runner := NewRunner(Config{
		RemoteURL: "https://vps.aris.ai:8080",
		Token:     "vps-token-123",
		NoBrowser: true, // Don't spawn real OS browser during test
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := runner.Run(ctx)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error running remote mode: %v", err)
	}
}

func TestDesktop_RunnerLocalMode(t *testing.T) {
	webCfg := web.Config{
		Host:     "127.0.0.1",
		Port:     0,
		AutoPort: true,
	}

	runner := NewRunner(Config{
		WebCfg:    webCfg,
		NoBrowser: true,
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := runner.Run(ctx)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Fatalf("unexpected error running local mode: %v", err)
	}
}
