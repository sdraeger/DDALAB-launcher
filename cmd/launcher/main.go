package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/ddalab/launcher/internal/app"
	"github.com/ddalab/launcher/internal/terminal"
	"github.com/ddalab/launcher/pkg/config"
)

// Version is set by build flags
var version = "dev"

func main() {
	// Handle CLI flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var forceMode = flag.String("mode", "", "Force operation mode: 'local', 'api', or 'auto'")
	var apiEndpoint = flag.String("api-endpoint", "", "Docker extension API endpoint (default: http://localhost:8080/api)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("DDALAB Launcher %s\n", version)
		fmt.Printf("Built with %s\n", runtime.Version())
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Check if we're running in a terminal
	if !terminal.IsTerminal() {
		// Try to relaunch in a terminal
		if err := terminal.RelaunchInTerminal(); err != nil {
			// If that fails, show a GUI error message
			terminal.ShowGUIError("Failed to open terminal",
				"DDALAB Launcher requires a terminal to run.\n\n"+
					"Please run this application from a terminal:\n"+
					"./ddalab-launcher")
			os.Exit(1)
		}
		// If relaunch succeeded, exit this instance
		os.Exit(0)
	}

	// Set terminal title
	if runtime.GOOS != "windows" {
		fmt.Print("\033]0;DDALAB Launcher\007")
	}

	// Set the version in the config package so it's available throughout the application
	config.SetVersion(version)

	launcher, err := app.NewLauncher()
	if err != nil {
		log.Fatalf("Failed to initialize launcher: %v", err)
	}

	// Apply CLI overrides if provided
	if err := applyModeOverrides(launcher, *forceMode, *apiEndpoint); err != nil {
		log.Fatalf("Failed to apply mode overrides: %v", err)
	}

	if err := launcher.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)

		// On error, wait for user input before closing
		fmt.Println("\nPress Enter to exit...")
		_, _ = fmt.Scanln()
		os.Exit(1)
	}
}

// applyModeOverrides applies CLI flag overrides to the launcher configuration
func applyModeOverrides(launcher *app.Launcher, forceMode, apiEndpoint string) error {
	configManager := launcher.GetConfigManager()

	// Override API endpoint if provided
	if apiEndpoint != "" {
		configManager.SetAPIEndpoint(apiEndpoint)
	}

	// Override operation mode if provided
	if forceMode != "" {
		var mode config.OperationMode
		switch strings.ToLower(forceMode) {
		case "local":
			mode = config.ModeLocal
		case "api":
			mode = config.ModeAPI
		case "auto":
			mode = config.ModeAuto
		default:
			return fmt.Errorf("invalid mode '%s'. Valid modes: local, api, auto", forceMode)
		}

		configManager.SetOperationMode(mode)

		// Save the configuration with overrides
		if err := configManager.Save(); err != nil {
			return fmt.Errorf("failed to save mode overrides: %w", err)
		}
	}

	return nil
}
