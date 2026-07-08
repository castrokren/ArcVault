//go:build linux

package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

const systemdTemplate = `[Unit]
Description={{.Description}}
After=network.target

[Service]
Type=simple
ExecStart={{.ExePath}}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
`

const unitPath = "/etc/systemd/system/arcvault-agent.service"

func install(exePath string) error {
	data := struct {
		Description string
		ExePath     string
	}{
		Description: AgentDescription,
		ExePath:     exePath,
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("unit").Parse(systemdTemplate))
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render unit file: %w", err)
	}

	if err := os.WriteFile(unitPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write unit file (run as root?): %w", err)
	}

	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", AgentServiceName},
	} {
		cmd := exec.Command("systemctl", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("systemctl %v failed: %w", args, err)
		}
	}

	fmt.Printf("Service %q installed and enabled.\n", AgentServiceName)
	fmt.Println("Start it with: sudo systemctl start arcvault-agent")
	return nil
}

func uninstall() error {
	for _, args := range [][]string{
		{"stop", AgentServiceName},
		{"disable", AgentServiceName},
	} {
		cmd := exec.Command("systemctl", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}

	exec.Command("systemctl", "daemon-reload").Run()

	fmt.Printf("Service %q uninstalled.\n", AgentServiceName)
	return nil
}
