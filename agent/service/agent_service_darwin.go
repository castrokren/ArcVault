//go:build darwin

package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.arcvault.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExePath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/arcvault-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/arcvault-agent-error.log</string>
</dict>
</plist>
`

const plistPath = "/Library/LaunchDaemons/com.arcvault.agent.plist"

func install(exePath string) error {
	data := struct{ ExePath string }{ExePath: exePath}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("plist").Parse(plistTemplate))
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render plist: %w", err)
	}

	if err := os.WriteFile(plistPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write plist (run as root?): %w", err)
	}

	cmd := exec.Command("launchctl", "load", plistPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launchctl load failed: %w", err)
	}

	fmt.Printf("Service %q installed and loaded.\n", AgentServiceName)
	fmt.Println("It will start automatically at boot.")
	return nil
}

func uninstall() error {
	exec.Command("launchctl", "unload", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	fmt.Printf("Service %q uninstalled.\n", AgentServiceName)
	return nil
}
