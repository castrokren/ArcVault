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

	// check if already installed
	existing, err := m.OpenService(CoordinatorServiceName)
	if err == nil {
		existing.Close()
		return fmt.Errorf("service %q is already installed", CoordinatorServiceName)
	}

	s, err := m.CreateService(
		CoordinatorServiceName,
		exePath,
		mgr.Config{
			DisplayName: CoordinatorDisplayName,
			Description: CoordinatorDescription,
			StartType:   mgr.StartAutomatic,
		},
		"start",
	)
	if err != nil {
		return fmt.Errorf("could not create service: %w", err)
	}
	defer s.Close()

	fmt.Printf("Service %q installed successfully.\n", CoordinatorServiceName)
	fmt.Println("Start it with: sc start arcvault-coordinator")
	fmt.Println("Or via Services (services.msc)")
	return nil
}

func uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(CoordinatorServiceName)
	if err != nil {
		return fmt.Errorf("service %q not found: %w", CoordinatorServiceName, err)
	}
	defer s.Close()

	if err := s.Delete(); err != nil {
		return fmt.Errorf("could not delete service: %w", err)
	}

	fmt.Printf("Service %q uninstalled successfully.\n", CoordinatorServiceName)
	return nil
}
