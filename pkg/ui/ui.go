package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/ddalab/launcher/pkg/config"
	"github.com/ddalab/launcher/pkg/detector"
	"github.com/manifoldco/promptui"
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
	config := ui.configManager.GetConfig()
	
	fmt.Printf("\n🚀 DDALAB Launcher v%s\n", config.Version)
	if config.DDALABPath != "" {
		fmt.Printf("📂 Installation: %s\n", config.DDALABPath)
	}

	menuManager := NewMenuManager(ui)
	options := menuManager.GetMainMenuOptions()
	
	action, err := menuManager.ShowMenu("What would you like to do?", options)
	if err != nil {
		return "", err
	}

	// Map actions back to original string format for compatibility
	actionMap := map[string]string{
		"start":     "Start DDALAB",
		"stop":      "Stop DDALAB",
		"restart":   "Restart DDALAB",
		"status":    "Check Status",
		"logs":      "View Logs",
		"configure": "Configure Installation",
		"backup":    "Backup Database",
		"update":    "Update DDALAB",
		"uninstall": "Uninstall DDALAB",
		"exit":      "Exit",
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

	prompt := promptui.Select{
		Label: "Select DDALAB installation",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", err
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

	prompt := promptui.Prompt{
		Label:    "Enter DDALAB installation path",
		Validate: validate,
		Default:  "~/DDALAB-setup",
	}

	result, err := prompt.Run()
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
	prompt := promptui.Select{
		Label: message,
		Items: []string{"Yes", "No"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		return false
	}

	return result == "Yes"
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
	
	prompt := promptui.Prompt{
		Label: message,
	}
	
	prompt.Run()
}