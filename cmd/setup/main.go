package main

import (
	"flag"
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
	// --install-dir lets the caller (Install-ArcVault.ps1) explicitly specify
	// where coordinator.exe / agent.exe live. This prevents config files being
	// written to the wrong directory when the wizard is run from a different
	// working directory during testing or re-runs.
	installDirFlag := flag.String("install-dir", "", "Directory containing coordinator.exe and agent.exe")
	flag.Parse()

	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║      ArcVault Setup Wizard         ║")
	fmt.Println("║          Version", Version[:6], "            ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println()

	// Resolve install path — explicit flag wins, fall back to wizard's own directory
	var installPath string
	if *installDirFlag != "" {
		abs, err := filepath.Abs(*installDirFlag)
		if err != nil {
			return fmt.Errorf("invalid --install-dir: %v", err)
		}
		installPath = abs
	} else {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine install path: %v", err)
		}
		installPath = filepath.Dir(exe)
	}

	fmt.Printf("Install directory: %s\n\n", installPath)

	// Verify coordinator.exe exists at install path — catch mismatches early
	coordExe := filepath.Join(installPath, "coordinator.exe")
	if _, err := os.Stat(coordExe); err != nil {
		return fmt.Errorf("coordinator.exe not found in %s — is --install-dir correct?", installPath)
	}

	// Step 1: Component Selection
	components, err := selectComponents()
	if err != nil {
		return fmt.Errorf("component selection failed: %v", err)
	}

	// Step 2: Collect Configuration
	config, err := gatherConfiguration(components)
	if err != nil {
		return fmt.Errorf("configuration gathering failed: %v", err)
	}

	// Step 3: Summary Review
	if err := reviewSummary(components, config); err != nil {
		return fmt.Errorf("setup cancelled by user")
	}

	// Step 4: Write config files
	if err := writeConfigurations(components, config, installPath); err != nil {
		return fmt.Errorf("failed to write configurations: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Configuration written successfully!")
	fmt.Println()

	// Step 5: Install and start services
	fmt.Println("Installing services...")
	if err := installServices(components, installPath); err != nil {
		return fmt.Errorf("failed to install services: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Setup complete!")
	fmt.Println()

	// Step 6: Open dashboard in browser (coordinator only)
	if components == ComponentCoordinator || components == ComponentBoth {
		url := fmt.Sprintf("http://localhost:%d", config.CoordinatorPort)
		fmt.Printf("Opening dashboard: %s\n", url)
		if err := OpenBrowser(url); err != nil {
			fmt.Printf("Note: Could not open browser automatically. Visit %s\n", url)
		}
	}

	return nil
}
