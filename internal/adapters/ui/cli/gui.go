package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"aris/internal/adapters/ui/desktop"
	"aris/internal/adapters/ui/web"
)

func (r *Runner) handleGUI(args []string) int {
	guiFlags := flag.NewFlagSet("gui", flag.ContinueOnError)

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

	hostFlag := guiFlags.String("host", defaultHost, "Host address to bind")
	_ = guiFlags.String("h", defaultHost, "Shorthand for host")
	portFlag := guiFlags.Int("port", defaultPort, "Port to listen on")
	_ = guiFlags.Int("p", defaultPort, "Shorthand for port")
	tokenFlag := guiFlags.String("token", defaultToken, "Authentication token for remote access")
	_ = guiFlags.String("t", defaultToken, "Shorthand for token")
	remoteFlag := guiFlags.String("remote", "", "Remote ARIS VPS URL (e.g. https://vps.aris.ai:8080)")
	_ = guiFlags.String("r", "", "Shorthand for remote")
	noBrowserFlag := guiFlags.Bool("no-browser", false, "Do not automatically launch OS browser window")

	if err := guiFlags.Parse(args); err != nil {
		return 1
	}

	webCfg := web.Config{
		Host:     *hostFlag,
		Port:     *portFlag,
		Token:    *tokenFlag,
		AutoPort: true,
	}

	desktopCfg := desktop.Config{
		RemoteURL: *remoteFlag,
		Token:     *tokenFlag,
		WebCfg:    webCfg,
		NoBrowser: *noBrowserFlag,
	}

	runner := desktop.NewRunner(desktopCfg, r.agent)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	select {
	case sig := <-sigCh:
		fmt.Printf("\n🛑 Received signal (%v), closing GUI...\n", sig)
		cancel()
		return 0
	case err := <-errCh:
		if err != nil {
			fmt.Printf("❌ GUI runner error: %v\n", err)
			return 1
		}
		return 0
	}
}
