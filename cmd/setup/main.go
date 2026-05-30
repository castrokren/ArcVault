package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Version is injected at build time via ldflags: -X main.Version=vX.Y.Z
var Version = "v1.1.0"

func main() {
	if err := RunSetupWizard(); err != nil {
		log.Fatalf("Setup wizard failed: %v", err)
	}
}

// RunSetupWizard orchestrates the entire setup flow
func RunSetupWizard() error {
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║      ArcVault Setup Wizard         ║")
	fmt.Println("║          Version", Version[:6], "            ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Component Selection
	components, err := selectComponents()
	if err != nil {
		return fmt.Errorf("component selection failed: %v", err)
	}

	// Step 2: Install path = directory containing this binary.
	// When launched from NSIS (C:\Program Files\ArcVault\arcvault-setup.exe),
	// this correctly resolves to C:\Program Files\ArcVault — the same
	// directory as coordinator.exe and agent.exe.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine install path: %v", err)
	}
	installPath := filepath.Dir(exe)

	// Step 3: Collect Configuration
	config, err := gatherConfiguration(components)
	if err != nil {
		return fmt.Errorf("configuration gathering failed: %v", err)
	}

	// Step 4: Summary Review
	if err := reviewSummary(components, config); err != nil {
		return fmt.Errorf("setup cancelled by user")
	}

	// Step 5: Write config files
	if err := writeConfigurations(components, config, installPath); err != nil {
		return fmt.Errorf("failed to write configurations: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Configuration written successfully!")
	fmt.Println()

	// Step 6: Install and start services
	fmt.Println("Installing services...")
	if err := installServices(components, installPath); err != nil {
		return fmt.Errorf("failed to install services: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Setup complete!")
	fmt.Println()

	// Step 7: Open dashboard in browser (coordinator only)
	if components == ComponentCoordinator || components == ComponentBoth {
		url := fmt.Sprintf("http://localhost:%d", config.CoordinatorPort)
		fmt.Printf("Opening dashboard: %s\n", url)
		if err := OpenBrowser(url); err != nil {
			fmt.Printf("Note: Could not open browser automatically. Visit %s\n", url)
		}
	}

	return nil
}
