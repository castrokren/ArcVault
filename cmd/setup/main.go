package main

import (
	"fmt"
	"log"
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

	// Step 2: Install Path (optional, use defaults)
	installPath := getInstallPath()

	// Step 3: Collect Configuration
	config, err := gatherConfiguration(components)
	if err != nil {
		return fmt.Errorf("configuration gathering failed: %v", err)
	}

	// Step 4: Summary Review
	if err := reviewSummary(components, config); err != nil {
		return fmt.Errorf("setup cancelled by user")
	}

	// Step 5: Write Configurations
	if err := writeConfigurations(components, config, installPath); err != nil {
		return fmt.Errorf("failed to write configurations: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Configuration completed successfully!")
	fmt.Println()

	// Step 6: Open Browser to Dashboard
	url := fmt.Sprintf("http://localhost:%d", config.CoordinatorPort)
	fmt.Printf("Opening dashboard: %s\n", url)
	if err := OpenBrowser(url); err != nil {
		fmt.Printf("Note: Could not open browser automatically. Visit %s in your browser.\n", url)
	}

	return nil
}
