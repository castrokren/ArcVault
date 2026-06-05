//go:build windows

package service

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc/mgr"
)

func install(exePath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// If the service already exists, delete it first so we can re-register cleanly.
	// This handles the case where the installer deleted the service but SCM hasn't
	// fully released it yet (exit code 1072 "marked for deletion").
	if existing, err := m.OpenService(CoordinatorServiceName); err == nil {
		_ = existing.Delete()
		existing.Close()
		// Wait up to 10 s for SCM to fully release the registration
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)
			if s, err := m.OpenService(CoordinatorServiceName); err != nil {
				// Service is gone — safe to create
				break
			} else {
				s.Close()
			}
		}
	}

	s, err := m.CreateService(
		CoordinatorServiceName,
		exePath,
		mgr.Config{
			DisplayName: CoordinatorDisplayName,
			Description: CoordinatorDescription,
			StartType:   mgr.StartAutomatic,
		},
		"run-service", // must use run-service, not start — SCM requires svc.Run()
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
