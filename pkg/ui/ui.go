package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/ddalab/launcher/pkg/config"
	"github.com/ddalab/launcher/pkg/detector"
)

// UI handles user interaction through prompts
type UI struct {
	configManager *config.ConfigManager
	detector      *detector.Detector
}

// NewUI creates a new UI instance
func NewUI(configManager *config.ConfigManager, detector *detector.Detector) *UI {
	return &UI{
		configManager: configManager,
		detector:      detector,
	}
}

// ShowWelcome displays the welcome message for first-time users
func (ui *UI) ShowWelcome() {
	fmt.Println("🚀 Welcome to DDALAB Launcher!")
	fmt.Println("This tool will help you manage your DDALAB installation easily.")
	fmt.Println("")
}

// ShowMainMenu displays the main menu for existing users
func (ui *UI) ShowMainMenu() (string, error) {
	return ui.ShowMainMenuWithStatus(nil)
}

// ShowMainMenuWithStatus displays the main menu with live status
func (ui *UI) ShowMainMenuWithStatus(statusMonitor any) (string, error) {
	config := ui.configManager.GetConfig()

	fmt.Printf("\n🚀 DDALAB Launcher %s\n", config.Version)
	if config.DDALABPath != "" {
		fmt.Printf("📂 Installation: %s\n", config.DDALABPath)
	}

	menuManager := NewMenuManager(ui)
	options := menuManager.GetMainMenuOptions()

	// Use status-aware menu if monitor is provided
	var action string
	var err error
	if statusMonitor != nil {
		if monitor, ok := statusMonitor.(interface{ FormatStatus() string }); ok {
			action, err = menuManager.ShowMenuWithStatus("What would you like to do?", options, monitor)
		} else {
			action, err = menuManager.ShowMenu("What would you like to do?", options)
		}
	} else {
		action, err = menuManager.ShowMenu("What would you like to do?", options)
	}
	if err != nil {
		return "", err
	}

	// Map actions back to original string format for compatibility
	actionMap := map[string]string{
		"start":         "Start DDALAB",
		"stop":          "Stop DDALAB",
		"restart":       "Restart DDALAB",
		"status":        "Check Status",
		"logs":          "View Logs",
		"bootstrap":     "Bootstrap DDALAB",
		"edit-config":   "Edit Configuration",
		"configure":     "Configure Installation",
		"backup":        "Backup Database",
		"update":        "Update DDALAB",
		"check-updates": "Check for Launcher Updates",
		"open-gui":      "Open GUI (Experimental)",
		"uninstall":     "Uninstall DDALAB",
		"exit":          "Exit",
	}

	if result, exists := actionMap[action]; exists {
		return result, nil
	}

	return action, nil
}

// SelectInstallation prompts user to select or configure an installation
func (ui *UI) SelectInstallation() (string, error) {
	// First, try to find existing installations
	installations, err := ui.detector.FindInstallations()
	if err != nil {
		return "", fmt.Errorf("error searching for installations: %w", err)
	}

	if len(installations) == 0 {
		return ui.configureNewInstallation()
	}

	// Show detected installations
	var items []string
	for _, install := range installations {
		status := "✅ Valid"
		if !install.Valid {
			status = "❌ Invalid"
		}
		items = append(items, fmt.Sprintf("%s (%s) - %s", install.Path, install.Version, status))
	}
	items = append(items, "➕ Configure new installation path")

	selectedItem, err := RunMenu("Select DDALAB installation", items)
	if err != nil {
		return "", err
	}

	// Find the index of the selected item
	index := -1
	for i, item := range items {
		if item == selectedItem {
			index = i
			break
		}
	}

	if index == -1 {
		return "", fmt.Errorf("invalid selection")
	}

	// If user selected "Configure new installation"
	if index == len(installations) {
		return ui.configureNewInstallation()
	}

	selectedInstall := installations[index]
	if !selectedInstall.Valid {
		fmt.Printf("⚠️  Warning: The selected installation appears to be invalid.\n")
		if !ui.confirmContinue("Do you want to continue anyway?") {
			return ui.SelectInstallation()
		}
	}

	return selectedInstall.Path, nil
}

// configureNewInstallation prompts user to enter a custom path
func (ui *UI) configureNewInstallation() (string, error) {
	validate := func(input string) error {
		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("path cannot be empty")
		}

		// Basic validation - check if path looks reasonable
		info := ui.detector.DetectInstallation(input)
		if !info.Valid {
			return fmt.Errorf("invalid DDALAB installation at %s", input)
		}

		return nil
	}

	result, err := RunPrompt("Enter DDALAB installation path", "~/DDALAB-setup", validate)
	if err != nil {
		return "", err
	}

	// Expand tilde
	if strings.HasPrefix(result, "~/") {
		homeDir, _ := os.UserHomeDir()
		result = strings.Replace(result, "~/", homeDir+"/", 1)
	}

	return result, nil
}

// ConfirmOperation asks user to confirm a potentially destructive operation
func (ui *UI) ConfirmOperation(operation string) bool {
	menuManager := NewMenuManager(ui)
	return menuManager.ShowConfirmation(fmt.Sprintf("Are you sure you want to %s?", operation))
}

// ShowServiceMenu displays the service management submenu
func (ui *UI) ShowServiceMenu() (string, error) {
	menuManager := NewMenuManager(ui)
	options := menuManager.GetServiceMenuOptions()
	return menuManager.ShowMenu("🔧 Service Management", options)
}

// ShowManagementMenu displays the system management submenu
func (ui *UI) ShowManagementMenu() (string, error) {
	menuManager := NewMenuManager(ui)
	options := menuManager.GetManagementMenuOptions()
	return menuManager.ShowMenu("⚙️ System Management", options)
}

// confirmContinue shows a yes/no prompt
func (ui *UI) confirmContinue(message string) bool {
	result, err := RunConfirm(message)
	if err != nil {
		return false
	}

	return result
}

// ShowProgress displays a progress message
func (ui *UI) ShowProgress(message string) {
	fmt.Printf("🔄 %s...\n", message)
}

// ShowSuccess displays a success message
func (ui *UI) ShowSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// ShowError displays an error message
func (ui *UI) ShowError(message string) {
	fmt.Printf("❌ Error: %s\n", message)
}

// ShowInfo displays an informational message
func (ui *UI) ShowInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// ShowWarning displays a warning message
func (ui *UI) ShowWarning(message string) {
	fmt.Printf("⚠️  Warning: %s\n", message)
}

// WaitForUser waits for user to press Enter
func (ui *UI) WaitForUser(message string) {
	if message == "" {
		message = "Press Enter to continue..."
	}

	_ = RunWait(message)
}
