package desktop

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// BuildLaunchURL appends authentication token to target URL if provided.
func BuildLaunchURL(baseURL, token string) string {
	if token == "" {
		return baseURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		if strings.Contains(baseURL, "?") {
			return baseURL + "&token=" + url.QueryEscape(token)
		}
		return baseURL + "?token=" + url.QueryEscape(token)
	}

	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

// OpenBrowser attempts to open the specified URL in the default web browser.
func OpenBrowser(targetURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", targetURL)
	case "darwin":
		cmd = exec.Command("open", targetURL)
	default:
		// Linux, BSD, Unix
		cmd = exec.Command("xdg-open", targetURL)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser (%s): %w", runtime.GOOS, err)
	}
	return nil
}
