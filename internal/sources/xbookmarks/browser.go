package xbookmarks

import (
	"os/exec"
	"runtime"
)

// openBrowser best-effort opens a URL in the user's default browser.
// Failure is non-fatal: the URL is also printed so the user can copy it.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
