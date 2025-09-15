package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ddalab/launcher/pkg/commands"
	"github.com/ddalab/launcher/pkg/config"
	"github.com/ddalab/launcher/pkg/status"
	"github.com/ddalab/launcher/pkg/updater"
)

// GUI represents the Fyne-based graphical interface
type GUI struct {
	app           fyne.App
	window        fyne.Window
	commander     *commands.Commander
	configMgr     *config.ConfigManager
	statusMonitor *status.Monitor

	// UI elements
	statusLabel   *widget.Label
	actionButtons *fyne.Container
	logOutput     *widget.Entry
}

// NewGUI creates a new GUI instance
func NewGUI(commander *commands.Commander, configMgr *config.ConfigManager, statusMonitor *status.Monitor) *GUI {
	fyneApp := app.NewWithID("com.ddalab.launcher")

	window := fyneApp.NewWindow("DDALAB Launcher")
	window.Resize(fyne.NewSize(600, 500))
	window.CenterOnScreen()

	return &GUI{
		app:           fyneApp,
		window:        window,
		commander:     commander,
		configMgr:     configMgr,
		statusMonitor: statusMonitor,
	}
}

// Show displays the GUI window
func (g *GUI) Show() {
	g.setupUI()
	g.startStatusUpdates()
	g.window.ShowAndRun()
}

// setupUI creates and arranges all UI elements
func (g *GUI) setupUI() {
	// Header with title and version
	title := widget.NewLabel("DDALAB Launcher")
	title.TextStyle = fyne.TextStyle{Bold: true}
	version := widget.NewLabel(fmt.Sprintf("Version: %s", config.GetVersion()))

	// Status section
	g.statusLabel = widget.NewLabel("Status: Checking...")
	g.statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Installation path
	pathLabel := widget.NewLabel(fmt.Sprintf("Installation: %s", g.configMgr.GetDDALABPath()))
	pathLabel.Wrapping = fyne.TextWrapWord

	// Action buttons
	g.actionButtons = container.NewVBox(
		g.createServiceButtons(),
		widget.NewSeparator(),
		g.createManagementButtons(),
		widget.NewSeparator(),
		g.createUtilityButtons(),
	)

	// Log output area
	g.logOutput = widget.NewMultiLineEntry()
	g.logOutput.SetPlaceHolder("Operation logs will appear here...")
	g.logOutput.Disable()
	logScroll := container.NewScroll(g.logOutput)
	logScroll.SetMinSize(fyne.NewSize(0, 150))

	// Layout everything
	header := container.NewVBox(title, version, widget.NewSeparator())
	statusSection := container.NewVBox(g.statusLabel, pathLabel, widget.NewSeparator())

	content := container.NewBorder(
		container.NewVBox(header, statusSection), // Top
		nil,                                      // Bottom
		nil,                                      // Left
		nil,                                      // Right
		container.NewHSplit(
			container.NewScroll(g.actionButtons),
			container.NewBorder(
				widget.NewLabel("Operation Log:"),
				nil, nil, nil,
				logScroll,
			),
		),
	)

	g.window.SetContent(content)
}

// createServiceButtons creates buttons for DDALAB service control
func (g *GUI) createServiceButtons() *widget.Card {
	startBtn := widget.NewButton("🚀 Start DDALAB", func() {
		g.executeOperation("Starting DDALAB", func(ctx context.Context) error {
			return g.commander.StartWithContext(ctx)
		})
	})
	startBtn.Importance = widget.HighImportance

	stopBtn := widget.NewButton("🛑 Stop DDALAB", func() {
		if g.confirmAction("Stop DDALAB", "Are you sure you want to stop all DDALAB services?") {
			g.executeOperation("Stopping DDALAB", func(ctx context.Context) error {
				return g.commander.Stop()
			})
		}
	})
	stopBtn.Importance = widget.MediumImportance

	restartBtn := widget.NewButton("🔄 Restart DDALAB", func() {
		if g.confirmAction("Restart DDALAB", "Are you sure you want to restart all DDALAB services?") {
			g.executeOperation("Restarting DDALAB", func(ctx context.Context) error {
				return g.commander.Restart()
			})
		}
	})

	statusBtn := widget.NewButton("📊 Check Status", func() {
		g.showDetailedStatus()
	})

	logsBtn := widget.NewButton("📋 View Logs", func() {
		g.executeOperation("Fetching logs", func(ctx context.Context) error {
			logs, err := g.commander.LogsWithContext(ctx)
			if err != nil {
				return err
			}
			g.showLogs(logs)
			return nil
		})
	})

	buttons := container.NewGridWithColumns(2, startBtn, stopBtn, restartBtn, statusBtn, logsBtn)
	return widget.NewCard("Service Control", "", buttons)
}

// createManagementButtons creates buttons for system management
func (g *GUI) createManagementButtons() *widget.Card {
	updateBtn := widget.NewButton("⬆️ Update DDALAB", func() {
		if g.confirmAction("Update DDALAB", "This will update DDALAB to the latest version. Continue?") {
			g.executeOperation("Updating DDALAB", func(ctx context.Context) error {
				return g.commander.UpdateWithContext(ctx)
			})
		}
	})

	backupBtn := widget.NewButton("💾 Backup Database", func() {
		g.executeOperation("Creating backup", func(ctx context.Context) error {
			return g.commander.Backup()
		})
	})

	configBtn := widget.NewButton("📝 Edit Configuration", func() {
		g.showInfo("Configuration", "Configuration editing in GUI mode is not yet implemented.\nPlease use the terminal interface for configuration editing.")
	})

	buttons := container.NewGridWithColumns(2, updateBtn, backupBtn, configBtn)
	return widget.NewCard("System Management", "", buttons)
}

// createUtilityButtons creates utility and launcher management buttons
func (g *GUI) createUtilityButtons() *widget.Card {
	launcherUpdateBtn := widget.NewButton("🔄 Check Launcher Updates", func() {
		g.checkLauncherUpdates()
	})

	uninstallBtn := widget.NewButton("🗑️ Uninstall DDALAB", func() {
		if g.confirmDangerousAction("Uninstall DDALAB",
			"⚠️ WARNING: This will completely remove DDALAB and all data!\n\nThis action cannot be undone. Are you absolutely sure?") {
			g.executeOperation("Uninstalling DDALAB", func(ctx context.Context) error {
				return g.commander.Uninstall()
			})
		}
	})
	uninstallBtn.Importance = widget.DangerImportance

	aboutBtn := widget.NewButton("ℹ️ About", func() {
		g.showAbout()
	})

	buttons := container.NewGridWithColumns(2, launcherUpdateBtn, aboutBtn, uninstallBtn)
	return widget.NewCard("Launcher Utilities", "", buttons)
}

// executeOperation runs an operation with progress indication and error handling
func (g *GUI) executeOperation(description string, operation func(context.Context) error) {
	g.logMessage(fmt.Sprintf("🔄 %s...", description))

	// Disable all buttons during operation
	g.setButtonsEnabled(false)
	defer g.setButtonsEnabled(true)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run operation
	if err := operation(ctx); err != nil {
		g.logMessage(fmt.Sprintf("❌ %s failed: %v", description, err))
		g.showError("Operation Failed", fmt.Sprintf("%s failed:\n\n%v", description, err))
	} else {
		g.logMessage(fmt.Sprintf("✅ %s completed successfully", description))
		g.showSuccess("Success", fmt.Sprintf("%s completed successfully!", description))
	}

	// Refresh status after operation
	g.statusMonitor.CheckNow()
}

// Helper methods for UI interactions
func (g *GUI) confirmAction(title, message string) bool {
	result := make(chan bool, 1)

	confirm := dialog.NewConfirm(title, message, func(confirmed bool) {
		result <- confirmed
	}, g.window)

	confirm.Show()
	return <-result
}

func (g *GUI) confirmDangerousAction(title, message string) bool {
	result := make(chan bool, 1)

	confirm := dialog.NewConfirm(title, message, func(confirmed bool) {
		result <- confirmed
	}, g.window)
	confirm.SetDismissText("Cancel")
	confirm.SetConfirmText("Yes, Uninstall")

	confirm.Show()
	return <-result
}

func (g *GUI) showInfo(title, message string) {
	dialog.ShowInformation(title, message, g.window)
}

func (g *GUI) showError(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), g.window)
}

func (g *GUI) showSuccess(title, message string) {
	dialog.ShowInformation(title, message, g.window)
}

func (g *GUI) logMessage(message string) {
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timestamp, message)

	if g.logOutput.Text != "" {
		g.logOutput.SetText(g.logOutput.Text + "\n" + logLine)
	} else {
		g.logOutput.SetText(logLine)
	}

	// Auto-scroll to bottom
	g.logOutput.CursorRow = strings.Count(g.logOutput.Text, "\n")
}

func (g *GUI) setButtonsEnabled(enabled bool) {
	// This is a simplified version - in a real implementation,
	// you'd want to track all buttons and disable/enable them
	// For now, this is a placeholder
}

// startStatusUpdates begins periodic status updates
func (g *GUI) startStatusUpdates() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				g.updateStatusDisplay()
			}
		}
	}()

	// Initial update
	g.updateStatusDisplay()
}

func (g *GUI) updateStatusDisplay() {
	if g.statusMonitor != nil {
		statusText := g.statusMonitor.FormatStatus()
		g.statusLabel.SetText(fmt.Sprintf("DDALAB Status: %s", statusText))
	}
}

func (g *GUI) showDetailedStatus() {
	running, err := g.commander.IsRunning()
	if err != nil {
		g.showError("Status Check Failed", fmt.Sprintf("Could not check status: %v", err))
		return
	}

	var message strings.Builder
	if running {
		message.WriteString("✅ DDALAB is running\n")
		message.WriteString("🌐 Access URL: https://localhost\n\n")

		// Get service health if possible
		services, err := g.commander.GetServiceHealth()
		if err == nil {
			message.WriteString("📊 Service Health:\n")
			for service, status := range services {
				icon := "🟢"
				if status != "Up" && status != "running" {
					icon = "🔴"
				}
				message.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, service, status))
			}
		}
	} else {
		message.WriteString("🔴 DDALAB is not running\n")
		message.WriteString("Use 'Start DDALAB' to launch services")
	}

	g.showInfo("DDALAB Status", message.String())
}

func (g *GUI) showLogs(logs string) {
	// Create a new window to show logs
	logWindow := g.app.NewWindow("DDALAB Logs")
	logWindow.Resize(fyne.NewSize(800, 600))

	logEntry := widget.NewMultiLineEntry()
	logEntry.SetText(logs)
	logEntry.Disable()

	closeBtn := widget.NewButton("Close", func() {
		logWindow.Close()
	})

	content := container.NewBorder(
		widget.NewLabel("Recent DDALAB Logs:"),
		closeBtn,
		nil, nil,
		container.NewScroll(logEntry),
	)

	logWindow.SetContent(content)
	logWindow.Show()
}

func (g *GUI) checkLauncherUpdates() {
	g.executeOperation("Checking for launcher updates", func(ctx context.Context) error {
		updaterInstance := updater.NewUpdater(config.GetVersion())
		updateInfo, err := updaterInstance.CheckForUpdates(ctx)
		if err != nil {
			return err
		}

		if !updateInfo.HasUpdate {
			g.showInfo("No Updates",
				fmt.Sprintf("You're running the latest version!\n\nCurrent: %s\nLatest: %s",
					updateInfo.CurrentVersion, updateInfo.LatestVersion))
			return nil
		}

		// Show update available dialog
		message := fmt.Sprintf("A new version is available!\n\nCurrent: %s\nLatest: %s\nReleased: %s\n\nWould you like to download and install it?",
			updateInfo.CurrentVersion,
			updateInfo.LatestVersion,
			updateInfo.PublishedAt.Format("January 2, 2006"))

		if g.confirmAction("Update Available", message) {
			g.logMessage("🔄 Downloading and installing launcher update...")
			err := updaterInstance.PerformUpdate(ctx, updateInfo.DownloadURL)
			if err != nil {
				return fmt.Errorf("failed to install update: %w", err)
			}

			g.showInfo("Update Complete",
				"Launcher update completed successfully!\n\nPlease restart the launcher to use the new version.")
		}

		return nil
	})
}

func (g *GUI) showAbout() {
	message := fmt.Sprintf(`DDALAB Launcher %s

A graphical and command-line tool for managing DDALAB installations.

DDALAB (Delay Differential Analysis Laboratory) is a scientific computing application for performing Delay Differential Analysis on EDF and ASCII files.

Features:
• Service Management (Start/Stop/Restart)
• Real-time Status Monitoring  
• Log Viewing
• Configuration Management
• Automatic Updates
• Database Backup

Platform: %s
`, config.GetVersion(), updater.GetPlatformString())

	g.showInfo("About DDALAB Launcher", message)
}
