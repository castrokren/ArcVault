package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the default web browser to the specified URL
// Works on Windows, macOS, and Linux
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use the Windows command to open a URL in the default browser
		cmd = exec.Command("cmd", "/c", "start", url)

	case "darwin":
		// macOS: use 'open' command
		cmd = exec.Command("open", url)

	case "linux":
		// Linux: try xdg-open first, fall back to other options
		// xdg-open is the standard freedesktop.org way
		cmd = exec.Command("xdg-open", url)

		// Check if xdg-open exists, otherwise try other options
		if err := cmd.Start(); err != nil {
			// Try alternative Linux browsers
			alternatives := []string{"firefox", "chromium", "chromium-browser", "google-chrome"}
			for _, browser := range alternatives {
				cmd = exec.Command(browser, url)
				if err := cmd.Start(); err == nil {
					return nil
				}
			}
			// If nothing worked, just inform the user
			return fmt.Errorf("could not find a suitable web browser")
		}
		return nil

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Start the command (don't wait for it to complete)
	return cmd.Start()
}
