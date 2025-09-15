package app

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/ddalab/launcher/pkg/commands"
	"github.com/ddalab/launcher/pkg/config"
	"github.com/ddalab/launcher/pkg/detector"
	"github.com/ddalab/launcher/pkg/interrupt"
	"github.com/ddalab/launcher/pkg/status"
	"github.com/ddalab/launcher/pkg/ui"
	"github.com/ddalab/launcher/pkg/updater"
)

// Launcher is the main application struct
type Launcher struct {
	configManager    *config.ConfigManager
	detector         *detector.Detector
	ui               *ui.UI
	commander        *commands.Commander
	interruptHandler *interrupt.Handler
	statusMonitor    *status.Monitor
}

// NewLauncher creates a new launcher instance
func NewLauncher() (*Launcher, error) {
	configManager, err := config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config manager: %w", err)
	}

	detector := detector.NewDetector()
	ui := ui.NewUI(configManager, detector)
	commander := commands.NewCommander(configManager)
	interruptHandler := interrupt.NewHandler()
	statusMonitor := status.NewMonitor(commander)

	return &Launcher{
		configManager:    configManager,
		detector:         detector,
		ui:               ui,
		commander:        commander,
		interruptHandler: interruptHandler,
		statusMonitor:    statusMonitor,
	}, nil
}

// Run starts the launcher application
func (l *Launcher) Run() error {
	// Check if this is the first run
	if l.configManager.IsFirstRun() {
		return l.runFirstTimeSetup()
	}

	// Show main menu for existing users
	return l.runMainLoop()
}

// runFirstTimeSetup handles the initial setup process
func (l *Launcher) runFirstTimeSetup() error {
	l.ui.ShowWelcome()

	// Detect or configure DDALAB installation
	ddalabPath, err := l.ui.SelectInstallation()
	if err != nil {
		return fmt.Errorf("installation selection failed: %w", err)
	}

	// Validate the installation
	l.ui.ShowProgress("Validating DDALAB installation")
	if err := l.detector.ValidateInstallation(ddalabPath); err != nil {
		l.ui.ShowError(fmt.Sprintf("Installation validation failed: %v", err))
		return err
	}

	// Save configuration
	l.configManager.SetDDALABPath(ddalabPath)
	if err := l.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	l.ui.ShowSuccess("DDALAB Launcher configured successfully!")
	l.ui.ShowInfo(fmt.Sprintf("Installation path: %s", ddalabPath))

	// Ask if user wants to start DDALAB now
	if l.ui.ConfirmOperation("start DDALAB now") {
		return l.handleStartCommand()
	}

	return nil
}

// runMainLoop handles the main menu loop with enhanced error handling
func (l *Launcher) runMainLoop() error {
	// Start status monitoring if DDALAB is configured
	if l.configManager.GetDDALABPath() != "" {
		l.statusMonitor.Start()
		defer l.statusMonitor.Stop()
	}

	// Check for launcher updates on startup (background check)
	l.checkForUpdatesOnStartup()

	for {
		// Clear screen for better UX
		fmt.Print("\033[2J\033[H")

		choice, err := l.ui.ShowMainMenuWithStatus(l.statusMonitor)
		if err != nil {
			// Handle user cancellation gracefully
			if err.Error() == "^C" || err.Error() == "interrupt" {
				l.ui.ShowInfo("Goodbye!")
				return nil
			}
			return fmt.Errorf("menu selection failed: %w", err)
		}

		// Exit the loop if user chose to exit
		if choice == "Exit" {
			l.ui.ShowInfo("Goodbye!")
			l.ui.WaitForUser("Press Enter to close...")
			break
		}

		// Handle the menu choice with error recovery
		if err := l.handleMenuChoice(choice); err != nil {
			l.ui.ShowError(err.Error())
			l.ui.WaitForUser("Press Enter to return to main menu...")
			continue
		}

		// Show success message and brief pause before returning to menu
		fmt.Println("\n✅ Operation completed successfully!")
		l.ui.WaitForUser("Press Enter to return to main menu...")
	}

	return nil
}

// executeWithInterrupt executes a function with interrupt handling
func (l *Launcher) executeWithInterrupt(operation string, fn func(ctx context.Context) error) error {
	fmt.Printf("ℹ️  Press Ctrl+C to cancel %s\n", operation)

	ctx, cancel := l.interruptHandler.WithCancellableContext(context.Background())
	defer cancel()

	err := fn(ctx)

	if interrupt.IsInterruptError(err) {
		l.ui.ShowWarning("Operation was cancelled")
		return nil // Don't treat cancellation as an error
	}

	if l.interruptHandler.WasInterrupted() {
		l.ui.ShowWarning("Operation was interrupted but may have completed")
		return nil
	}

	return err
}

// handleMenuChoice processes the user's menu selection
func (l *Launcher) handleMenuChoice(choice string) error {
	fmt.Printf("\n🔄 Processing: %s\n", choice)
	fmt.Println("═════════════════════════════════════")

	switch choice {
	case "Start DDALAB":
		return l.handleStartCommand()
	case "Stop DDALAB":
		return l.handleStopCommand()
	case "Restart DDALAB":
		return l.handleRestartCommand()
	case "Check Status":
		return l.handleStatusCommand()
	case "View Logs":
		return l.handleLogsCommand()
	case "Edit Configuration":
		return l.handleEditConfigCommand()
	case "Configure Installation":
		return l.handleConfigureCommand()
	case "Backup Database":
		return l.handleBackupCommand()
	case "Update DDALAB":
		return l.handleUpdateCommand()
	case "Check for Launcher Updates":
		return l.handleCheckUpdatesCommand()
	case "Uninstall DDALAB":
		return l.handleUninstallCommand()
	case "Exit":
		// This case is handled in the main loop
		return nil
	default:
		return fmt.Errorf("unknown menu choice: %s", choice)
	}
}

// handleStartCommand starts DDALAB services
func (l *Launcher) handleStartCommand() error {
	// Check if already running
	running, err := l.commander.IsRunning()
	if err != nil {
		l.ui.ShowWarning(fmt.Sprintf("Could not check running status: %v", err))
	} else if running {
		l.ui.ShowInfo("DDALAB is already running")
		return nil
	}

	return l.executeWithInterrupt("starting DDALAB", func(ctx context.Context) error {
		l.ui.ShowProgress("Starting DDALAB services")
		if err := l.commander.StartWithContext(ctx); err != nil {
			return fmt.Errorf("failed to start DDALAB: %w", err)
		}

		l.ui.ShowSuccess("DDALAB started successfully!")
		l.ui.ShowInfo("Access DDALAB at: https://localhost")

		// Refresh status after starting
		l.statusMonitor.CheckNow()
		return nil
	})
}

// handleStopCommand stops DDALAB services
func (l *Launcher) handleStopCommand() error {
	if !l.ui.ConfirmOperation("stop DDALAB") {
		return nil
	}

	l.ui.ShowProgress("Stopping DDALAB services")
	if err := l.commander.Stop(); err != nil {
		return fmt.Errorf("failed to stop DDALAB: %w", err)
	}

	l.ui.ShowSuccess("DDALAB stopped successfully!")

	// Refresh status after stopping
	l.statusMonitor.CheckNow()
	return nil
}

// handleRestartCommand restarts DDALAB services
func (l *Launcher) handleRestartCommand() error {
	if !l.ui.ConfirmOperation("restart DDALAB") {
		return nil
	}

	l.ui.ShowProgress("Restarting DDALAB services")
	if err := l.commander.Restart(); err != nil {
		return fmt.Errorf("failed to restart DDALAB: %w", err)
	}

	l.ui.ShowSuccess("DDALAB restarted successfully!")

	// Refresh status after restarting
	l.statusMonitor.CheckNow()
	return nil
}

// handleStatusCommand shows DDALAB service status
func (l *Launcher) handleStatusCommand() error {
	l.ui.ShowProgress("Checking DDALAB status")

	// Check if services are running
	running, err := l.commander.IsRunning()
	if err != nil {
		return fmt.Errorf("failed to check running status: %w", err)
	}

	if running {
		l.ui.ShowSuccess("DDALAB is running")
		l.ui.ShowInfo("Access URL: https://localhost")

		// Get detailed service health
		services, err := l.commander.GetServiceHealth()
		if err != nil {
			l.ui.ShowWarning(fmt.Sprintf("Could not get detailed status: %v", err))
		} else {
			fmt.Println("\n📊 Service Status:")
			for service, status := range services {
				statusIcon := "🟢"
				if status != "Up" && status != "running" {
					statusIcon = "🔴"
				}
				fmt.Printf("  %s %s: %s\n", statusIcon, service, status)
			}
		}
	} else {
		l.ui.ShowInfo("DDALAB is not running")
		l.ui.ShowInfo("Use 'Start DDALAB' to launch services")
	}

	return nil
}

// handleLogsCommand shows DDALAB service logs
func (l *Launcher) handleLogsCommand() error {
	return l.executeWithInterrupt("fetching logs", func(ctx context.Context) error {
		l.ui.ShowProgress("Fetching DDALAB logs")

		logs, err := l.commander.LogsWithContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		fmt.Println("\n📋 === DDALAB Recent Logs ===")
		fmt.Println(logs)
		fmt.Println("═══════════════════════════════")
		l.ui.ShowInfo("To view live logs, use: docker-compose logs -f")

		return nil
	})
}

// handleConfigureCommand reconfigures the DDALAB installation
func (l *Launcher) handleConfigureCommand() error {
	l.ui.ShowInfo("Reconfiguring DDALAB installation...")

	ddalabPath, err := l.ui.SelectInstallation()
	if err != nil {
		return fmt.Errorf("installation selection failed: %w", err)
	}

	// Validate the new installation
	l.ui.ShowProgress("Validating new installation")
	if err := l.detector.ValidateInstallation(ddalabPath); err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}

	// Save new configuration
	l.configManager.SetDDALABPath(ddalabPath)
	if err := l.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	l.ui.ShowSuccess("Configuration updated successfully!")
	l.ui.ShowInfo(fmt.Sprintf("New installation path: %s", ddalabPath))

	// Start status monitoring now that we have a valid installation
	if !l.statusMonitor.IsRunning() {
		l.statusMonitor.Start()
	}
	l.statusMonitor.CheckNow()

	return nil
}

// handleBackupCommand creates a database backup
func (l *Launcher) handleBackupCommand() error {
	l.ui.ShowProgress("Creating database backup")

	if err := l.commander.Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	l.ui.ShowSuccess("Database backup created successfully!")
	return nil
}

// handleUpdateCommand updates DDALAB to the latest version
func (l *Launcher) handleUpdateCommand() error {
	if !l.ui.ConfirmOperation("update DDALAB to the latest version") {
		return nil
	}

	return l.executeWithInterrupt("updating DDALAB", func(ctx context.Context) error {
		l.ui.ShowProgress("Updating DDALAB")
		l.ui.ShowInfo("This may take a few minutes...")

		if err := l.commander.UpdateWithContext(ctx); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		l.ui.ShowSuccess("DDALAB updated successfully!")
		return nil
	})
}

// handleUninstallCommand removes DDALAB installation
func (l *Launcher) handleUninstallCommand() error {
	l.ui.ShowWarning("This will stop all DDALAB services and remove all data!")

	if !l.ui.ConfirmOperation("completely uninstall DDALAB") {
		return nil
	}

	// Double confirmation for destructive operation
	if !l.ui.ConfirmOperation("permanently delete all DDALAB data") {
		return nil
	}

	l.ui.ShowProgress("Uninstalling DDALAB")

	if err := l.commander.Uninstall(); err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}

	l.ui.ShowSuccess("DDALAB uninstalled successfully!")
	l.ui.ShowInfo("You can safely delete the DDALAB-setup directory if no longer needed")

	return nil
}

// handleEditConfigCommand opens the configuration editor
func (l *Launcher) handleEditConfigCommand() error {
	// Find the .env file in the DDALAB installation
	ddalabPath := l.configManager.GetDDALABPath()
	envPath, err := config.GetEnvFilePath(ddalabPath)
	if err != nil {
		if strings.Contains(err.Error(), ".env.example exists") {
			l.ui.ShowWarning("No .env file found!")
			l.ui.ShowInfo("You need to create a .env file first from the .env.example template.")

			examplePath := strings.Replace(envPath, ".env", ".env.example", 1)
			l.ui.ShowInfo(fmt.Sprintf("Example file location: %s", examplePath))

			if l.ui.ConfirmOperation("copy .env.example to .env now") {
				if copyErr := config.CopyFile(examplePath, envPath); copyErr != nil {
					return fmt.Errorf("failed to copy .env.example: %w", copyErr)
				}
				l.ui.ShowSuccess("Created .env file from template")
			} else {
				return nil
			}
		} else {
			return fmt.Errorf("could not find .env file: %w", err)
		}
	}

	l.ui.ShowInfo(fmt.Sprintf("Opening configuration editor for: %s", envPath))
	l.ui.ShowInfo("Use arrow keys to navigate, Enter to edit, / to search, s to save, q to quit")
	l.ui.WaitForUser("Press Enter to open editor...")

	// Clear screen before launching editor
	fmt.Print("\033[2J\033[H")

	// Run the configuration editor
	if err := config.RunConfigEditor(envPath); err != nil {
		return fmt.Errorf("configuration editor failed: %w", err)
	}

	// Clear screen and show completion message
	fmt.Print("\033[2J\033[H")
	l.ui.ShowSuccess("Configuration editor closed")
	l.ui.ShowInfo("If you made changes, you may need to restart DDALAB for them to take effect")

	return nil
}

// handleCheckUpdatesCommand checks for launcher updates
func (l *Launcher) handleCheckUpdatesCommand() error {
	return l.executeWithInterrupt("checking for updates", func(ctx context.Context) error {
		l.ui.ShowProgress("Checking for launcher updates")

		// Create updater - use the actual binary version, not the config version
		currentVersion := config.GetVersion()
		updaterInstance := updater.NewUpdater(currentVersion)

		// Check for updates
		updateInfo, err := updaterInstance.CheckForUpdates(ctx)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		// Record the check time
		l.configManager.SetLastUpdateCheck(time.Now())
		if err := l.configManager.Save(); err != nil {
			l.ui.ShowWarning(fmt.Sprintf("Failed to save last update check time: %v", err))
		}

		if !updateInfo.HasUpdate {
			l.ui.ShowSuccess("You're running the latest version!")
			l.ui.ShowInfo(fmt.Sprintf("Current version: %s", updateInfo.CurrentVersion))
			l.ui.ShowInfo(fmt.Sprintf("Latest version: %s", updateInfo.LatestVersion))
			l.ui.ShowInfo(fmt.Sprintf("Platform: %s", updater.GetPlatformString()))
			return nil
		}

		// Show update information
		l.ui.ShowSuccess("A new version is available!")
		l.ui.ShowInfo(fmt.Sprintf("Current version: %s", updateInfo.CurrentVersion))
		l.ui.ShowInfo(fmt.Sprintf("Latest version: %s", updateInfo.LatestVersion))
		l.ui.ShowInfo(fmt.Sprintf("Released: %s", updateInfo.PublishedAt.Format("January 2, 2006")))

		if updateInfo.Size > 0 {
			l.ui.ShowInfo(fmt.Sprintf("Download size: %s", updater.FormatSize(updateInfo.Size)))
		}

		if updateInfo.ReleaseNotes != "" {
			fmt.Println("\n📋 Release Notes:")
			fmt.Println(updateInfo.ReleaseNotes)
		}

		if updateInfo.DownloadURL == "" {
			l.ui.ShowWarning("No download available for your platform")
			l.ui.ShowInfo(fmt.Sprintf("Platform: %s", updater.GetPlatformString()))
			return nil
		}

		// Ask user if they want to update
		if !l.ui.ConfirmOperation("download and install this update") {
			l.ui.ShowInfo("Update cancelled")
			return nil
		}

		return l.performLauncherUpdate(ctx, updaterInstance, updateInfo)
	})
}

// performLauncherUpdate downloads and applies the launcher update
func (l *Launcher) performLauncherUpdate(ctx context.Context, updaterInstance *updater.Updater, updateInfo *updater.UpdateInfo) error {
	l.ui.ShowProgress("Downloading update")
	l.ui.ShowInfo("This may take a moment...")

	err := updaterInstance.PerformUpdate(ctx, updateInfo.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	l.ui.ShowSuccess("Update completed successfully!")
	l.ui.ShowInfo(fmt.Sprintf("Updated to version %s", updateInfo.LatestVersion))

	// Platform-specific restart instructions
	switch runtime.GOOS {
	case "windows":
		l.ui.ShowInfo("The launcher will be replaced automatically when you exit.")
		l.ui.ShowInfo("Please close this window and start the launcher again.")
	default:
		l.ui.ShowInfo("The update has been applied to the binary.")
		l.ui.ShowInfo("Please restart the launcher to use the new version.")
	}

	// Update the version in config
	l.configManager.GetConfig().Version = updateInfo.LatestVersion
	if err := l.configManager.Save(); err != nil {
		l.ui.ShowWarning(fmt.Sprintf("Failed to save version info: %v", err))
	}

	return nil
}

// checkForUpdatesOnStartup performs automatic update checks if enabled
func (l *Launcher) checkForUpdatesOnStartup() {
	// Skip if auto-update is disabled or not time to check
	if !l.configManager.ShouldCheckForUpdates() {
		return
	}

	// Show brief message about background check
	l.ui.ShowInfo("Checking for launcher updates...")

	// Create a context with timeout for background check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the actual binary version, not the config version
	currentVersion := config.GetVersion()
	updaterInstance := updater.NewUpdater(currentVersion)

	updateInfo, err := updaterInstance.CheckForUpdates(ctx)
	if err != nil {
		// Silently fail for background checks - don't disturb user experience
		l.configManager.SetLastUpdateCheck(time.Now())
		_ = l.configManager.Save()
		return
	}

	// Record the check time
	l.configManager.SetLastUpdateCheck(time.Now())
	_ = l.configManager.Save()

	if updateInfo.HasUpdate {
		l.ui.ShowInfo(fmt.Sprintf("📦 Update available: %s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion))
		l.ui.ShowInfo("Use 'Check for Launcher Updates' from the menu to install")
	}
}
