package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/ddalab/launcher/internal/app"
	"github.com/ddalab/launcher/internal/terminal"
)

// Version is set by build flags
var version = "dev"

func main() {
	// Handle version flag
	var showVersion = flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("DDALAB Launcher v%s\n", version)
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

	launcher, err := app.NewLauncher()
	if err != nil {
		log.Fatalf("Failed to initialize launcher: %v", err)
	}

	if err := launcher.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)

		// On error, wait for user input before closing
		fmt.Println("\nPress Enter to exit...")
		_, _ = fmt.Scanln()
		os.Exit(1)
	}
}
