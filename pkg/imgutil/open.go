package imgutil

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenInViewer attempts to open the image file using the operating system's default viewer.
func OpenInViewer(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", filePath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open file in viewer (%s): %w", runtime.GOOS, err)
	}
	return nil
}
