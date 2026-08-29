package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"aris/internal/adapters/ui/web"
)

func (r *Runner) handleServe(args []string) int {
	serveFlags := flag.NewFlagSet("serve", flag.ContinueOnError)

	defaultHost := "127.0.0.1"
	if h := os.Getenv("ARIS_WEB_HOST"); h != "" {
		defaultHost = h
	}

	defaultPort := 8080
	if p := os.Getenv("ARIS_WEB_PORT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			defaultPort = val
		}
	}

	defaultToken := os.Getenv("ARIS_WEB_TOKEN")

	hostFlag := serveFlags.String("host", defaultHost, "Host address to bind")
	_ = serveFlags.String("h", defaultHost, "Shorthand for host")
	portFlag := serveFlags.Int("port", defaultPort, "Port to listen on")
	_ = serveFlags.Int("p", defaultPort, "Shorthand for port")
	tokenFlag := serveFlags.String("token", defaultToken, "Authentication token for remote access")
	_ = serveFlags.String("t", defaultToken, "Shorthand for token")
	autoPortFlag := serveFlags.Bool("auto-port", true, "Automatically fallback to next available port if occupied")

	if err := serveFlags.Parse(args); err != nil {
		return 1
	}

	cfg := web.Config{
		Host:     *hostFlag,
		Port:     *portFlag,
		Token:    *tokenFlag,
		AutoPort: *autoPortFlag,
	}

	server := web.NewServer(cfg, r.agent)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Wait for server to bind and print message
	for i := 0; i < 50; i++ {
		if server.URL() != "" {
			fmt.Printf("🚀 ARIS Web Server running at: %s\n", server.URL())
			if cfg.Token != "" {
				fmt.Printf("🔒 Authentication enabled (Token: %s)\n", cfg.Token)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case sig := <-sigCh:
		fmt.Printf("\n🛑 Received signal (%v), shutting down web server...\n", sig)
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
		return 0
	case err := <-errCh:
		if err != nil {
			fmt.Printf("❌ Web server error: %v\n", err)
			return 1
		}
		return 0
	}
}
