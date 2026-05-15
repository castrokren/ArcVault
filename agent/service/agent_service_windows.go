//go:build windows

package service

import (
	"fmt"

	"golang.org/x/sys/windows/svc/mgr"
)

func install(exePath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	existing, err := m.OpenService(AgentServiceName)
	if err == nil {
		existing.Close()
		return fmt.Errorf("service %q is already installed", AgentServiceName)
	}

	s, err := m.CreateService(
		AgentServiceName,
		exePath,
		mgr.Config{
			DisplayName: AgentDisplayName,
			Description: AgentDescription,
			StartType:   mgr.StartAutomatic,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create service: %w", err)
	}
	defer s.Close()

	fmt.Printf("Service %q installed successfully.\n", AgentServiceName)
	fmt.Println("Start it with: sc start arcvault-agent")
	return nil
}

func uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(AgentServiceName)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", AgentServiceName, err)
	}
	defer s.Close()

	if err := s.Delete(); err != nil {
		return fmt.Errorf("could not delete service: %w", err)
	}

	fmt.Printf("Service %q uninstalled successfully.\n", AgentServiceName)
	return nil
}
