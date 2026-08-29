package desktop

import (
	"context"
	"fmt"
	"log"
	"time"

	"aris/internal/adapters/ui/web"
	"aris/internal/core/services"
)

// Config configures the desktop runner.
type Config struct {
	RemoteURL string
	Token     string
	WebCfg    web.Config
	NoBrowser bool
}

// Runner manages desktop and browser window lifecycles.
type Runner struct {
	cfg   Config
	agent *services.AgentService
}

// NewRunner creates a new desktop Runner.
func NewRunner(cfg Config, agent *services.AgentService) *Runner {
	return &Runner{
		cfg:   cfg,
		agent: agent,
	}
}

// Run executes the desktop launcher.
func (r *Runner) Run(ctx context.Context) error {
	if r.cfg.RemoteURL != "" {
		// Remote VPS Mode
		launchURL := BuildLaunchURL(r.cfg.RemoteURL, r.cfg.Token)
		fmt.Printf("🌐 Connecting to remote ARIS VPS: %s\n", launchURL)

		if !r.cfg.NoBrowser {
			log.Println("Native webview unavailable; falling back to default web browser.")
			if err := OpenBrowser(launchURL); err != nil {
				fmt.Printf("⚠️ Failed to open default browser automatically: %v\n", err)
				fmt.Printf("👉 Please open the following URL in your browser: %s\n", launchURL)
			}
		}

		<-ctx.Done()
		return nil
	}

	// Local In-Process Web Server Mode
	server := web.NewServer(r.cfg.WebCfg, r.agent)

	srvErrCh := make(chan error, 1)
	go func() {
		srvErrCh <- server.Start(ctx)
	}()

	// Wait for server to bind
	var targetURL string
	for i := 0; i < 50; i++ {
		if server.URL() != "" {
			targetURL = server.URL()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if targetURL == "" {
		return fmt.Errorf("timed out waiting for web server to start")
	}

	launchURL := BuildLaunchURL(targetURL, r.cfg.Token)
	fmt.Printf("🚀 ARIS Local Web Server listening at: %s\n", launchURL)

	if !r.cfg.NoBrowser {
		log.Println("Native webview unavailable; falling back to default web browser.")
		if err := OpenBrowser(launchURL); err != nil {
			fmt.Printf("⚠️ Failed to open default browser automatically: %v\n", err)
			fmt.Printf("👉 Please open the following URL in your browser: %s\n", launchURL)
		}
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-srvErrCh:
		return err
	}
}
