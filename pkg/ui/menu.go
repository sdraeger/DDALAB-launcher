package ui

import (
	"fmt"
)

// MenuOption represents a menu choice with associated data
type MenuOption struct {
	Label       string
	Description string
	Action      string
	Icon        string
}

// MenuManager handles menu navigation and display
type MenuManager struct {
	ui *UI
}

// NewMenuManager creates a new menu manager
func NewMenuManager(ui *UI) *MenuManager {
	return &MenuManager{ui: ui}
}

// ShowMenu displays a menu with the given options and returns the selected action
func (m *MenuManager) ShowMenu(title string, options []MenuOption) (string, error) {
	items := make([]string, len(options))
	for i, option := range options {
		if option.Icon != "" {
			items[i] = fmt.Sprintf("%s %s", option.Icon, option.Label)
		} else {
			items[i] = option.Label
		}

		if option.Description != "" {
			items[i] += fmt.Sprintf(" - %s", option.Description)
		}
	}

	selectedItem, err := RunMenu(title, items)
	if err != nil {
		return "", err
	}

	// Find the corresponding action
	for i, item := range items {
		if item == selectedItem {
			return options[i].Action, nil
		}
	}

	return "", fmt.Errorf("action not found for selected item")
}

// ShowMenuWithStatus displays a menu with live status updates
func (m *MenuManager) ShowMenuWithStatus(title string, options []MenuOption, statusMonitor interface{ FormatStatus() string }) (string, error) {
	items := make([]string, len(options))
	for i, option := range options {
		if option.Icon != "" {
			items[i] = fmt.Sprintf("%s %s", option.Icon, option.Label)
		} else {
			items[i] = option.Label
		}

		if option.Description != "" {
			items[i] += fmt.Sprintf(" - %s", option.Description)
		}
	}

	selectedItem, err := RunMenuWithStatus(title, items, statusMonitor)
	if err != nil {
		return "", err
	}

	// Find the corresponding action
	for i, item := range items {
		if item == selectedItem {
			return options[i].Action, nil
		}
	}

	return "", fmt.Errorf("invalid selection")
}

// GetMainMenuOptions returns the standard main menu options
func (m *MenuManager) GetMainMenuOptions() []MenuOption {
	return []MenuOption{
		{Label: "Start DDALAB", Action: "start", Icon: "🚀", Description: "Start all DDALAB services"},
		{Label: "Stop DDALAB", Action: "stop", Icon: "🛑", Description: "Stop all DDALAB services"},
		{Label: "Restart DDALAB", Action: "restart", Icon: "🔄", Description: "Restart all DDALAB services"},
		{Label: "Check Status", Action: "status", Icon: "📊", Description: "Check service status and health"},
		{Label: "View Logs", Action: "logs", Icon: "📋", Description: "View recent service logs"},
		{Label: "Bootstrap DDALAB", Action: "bootstrap", Icon: "🔧", Description: "Bootstrap DDALAB services when API is unavailable"},
		{Label: "Edit Configuration", Action: "edit-config", Icon: "📝", Description: "Edit environment variables and settings"},
		{Label: "Configure Installation", Action: "configure", Icon: "⚙️", Description: "Change DDALAB installation path"},
		{Label: "Backup Database", Action: "backup", Icon: "💾", Description: "Create database backup"},
		{Label: "Update DDALAB", Action: "update", Icon: "⬆️", Description: "Update to latest version"},
		{Label: "Check for Launcher Updates", Action: "check-updates", Icon: "🔄", Description: "Check for launcher updates"},
		{Label: "Uninstall DDALAB", Action: "uninstall", Icon: "🗑️", Description: "Remove DDALAB completely"},
		{Label: "Exit", Action: "exit", Icon: "👋", Description: "Exit the launcher"},
	}
}

// GetMainMenuOptionsWithBootstrapContext returns menu options adapted for bootstrap context
func (m *MenuManager) GetMainMenuOptionsWithBootstrapContext(canBootstrap bool, isAPIMode bool) []MenuOption {
	options := []MenuOption{
		{Label: "Start DDALAB", Action: "start", Icon: "🚀", Description: "Start all DDALAB services"},
		{Label: "Stop DDALAB", Action: "stop", Icon: "🛑", Description: "Stop all DDALAB services"},
		{Label: "Restart DDALAB", Action: "restart", Icon: "🔄", Description: "Restart all DDALAB services"},
		{Label: "Check Status", Action: "status", Icon: "📊", Description: "Check service status and health"},
		{Label: "View Logs", Action: "logs", Icon: "📋", Description: "View recent service logs"},
	}

	// Add bootstrap option only if not in API mode and bootstrap is available
	if !isAPIMode && canBootstrap {
		options = append(options, MenuOption{
			Label:       "Bootstrap DDALAB",
			Action:      "bootstrap",
			Icon:        "🔧",
			Description: "Bootstrap DDALAB services (minimal setup)",
		})
	}

	// Add common options
	options = append(options, []MenuOption{
		{Label: "Edit Configuration", Action: "edit-config", Icon: "📝", Description: "Edit environment variables and settings"},
		{Label: "Configure Installation", Action: "configure", Icon: "⚙️", Description: "Change DDALAB installation path"},
		{Label: "Backup Database", Action: "backup", Icon: "💾", Description: "Create database backup"},
		{Label: "Update DDALAB", Action: "update", Icon: "⬆️", Description: "Update to latest version"},
		{Label: "Check for Launcher Updates", Action: "check-updates", Icon: "🔄", Description: "Check for launcher updates"},
		{Label: "Uninstall DDALAB", Action: "uninstall", Icon: "🗑️", Description: "Remove DDALAB completely"},
		{Label: "Exit", Action: "exit", Icon: "👋", Description: "Exit the launcher"},
	}...)

	return options
}

// GetManagementMenuOptions returns management-specific menu options
func (m *MenuManager) GetManagementMenuOptions() []MenuOption {
	return []MenuOption{
		{Label: "Configure Installation", Action: "configure", Icon: "⚙️"},
		{Label: "Backup Database", Action: "backup", Icon: "💾"},
		{Label: "Update DDALAB", Action: "update", Icon: "⬆️"},
		{Label: "Uninstall DDALAB", Action: "uninstall", Icon: "🗑️"},
		{Label: "Back to Main Menu", Action: "back", Icon: "⬅️"},
	}
}

// GetServiceMenuOptions returns service control menu options
func (m *MenuManager) GetServiceMenuOptions() []MenuOption {
	return []MenuOption{
		{Label: "Start DDALAB", Action: "start", Icon: "🚀"},
		{Label: "Stop DDALAB", Action: "stop", Icon: "🛑"},
		{Label: "Restart DDALAB", Action: "restart", Icon: "🔄"},
		{Label: "Check Status", Action: "status", Icon: "📊"},
		{Label: "View Logs", Action: "logs", Icon: "📋"},
		{Label: "Back to Main Menu", Action: "back", Icon: "⬅️"},
	}
}

// ShowConfirmation shows a confirmation dialog and returns true if confirmed
func (m *MenuManager) ShowConfirmation(message string) bool {
	result, err := RunConfirm(message)
	if err != nil {
		return false
	}

	return result
}

// ShowSubMenu displays a submenu and handles navigation
func (m *MenuManager) ShowSubMenu(title string, options []MenuOption, handler func(string) error) error {
	for {
		action, err := m.ShowMenu(title, options)
		if err != nil {
			return err
		}

		if action == "back" {
			return nil
		}

		if err := handler(action); err != nil {
			m.ui.ShowError(err.Error())
			m.ui.WaitForUser("")
			continue
		}
	}
}
