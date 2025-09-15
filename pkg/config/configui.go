package config

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	requiredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	secretStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Margin(1, 0, 0, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Margin(1, 0)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
)

// ConfigEditorModel represents the configuration editor state
type ConfigEditorModel struct {
	config         *EnvConfig
	cursor         int
	editMode       bool
	editingValue   string
	editingKey     string
	searchMode     bool
	searchTerm     string
	filteredVars   []EnvVar
	originalVars   []EnvVar
	width          int
	height         int
	saved          bool
	message        string
	showSecrets    bool
}

// NewConfigEditor creates a new configuration editor model
func NewConfigEditor(config *EnvConfig) *ConfigEditorModel {
	model := &ConfigEditorModel{
		config:       config,
		originalVars: make([]EnvVar, len(config.Variables)),
		filteredVars: config.Variables,
		width:        120,
		height:       30,
	}
	
	// Create a copy of original variables for comparison
	copy(model.originalVars, config.Variables)
	
	return model
}

// Init initializes the model
func (m *ConfigEditorModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *ConfigEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.editMode {
			return m.handleEditMode(msg)
		}
		
		if m.searchMode {
			return m.handleSearchMode(msg)
		}

		return m.handleNormalMode(msg)
	}

	return m, nil
}

// handleNormalMode handles key presses in normal navigation mode
func (m *ConfigEditorModel) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.filteredVars)-1 {
			m.cursor++
		}

	case "pgup":
		m.cursor = max(0, m.cursor-10)

	case "pgdown":
		m.cursor = min(len(m.filteredVars)-1, m.cursor+10)

	case "home":
		m.cursor = 0

	case "end":
		m.cursor = len(m.filteredVars) - 1

	case "enter", " ":
		if len(m.filteredVars) > 0 {
			m.editMode = true
			m.editingKey = m.filteredVars[m.cursor].Key
			m.editingValue = m.filteredVars[m.cursor].Value
		}

	case "/":
		m.searchMode = true
		m.searchTerm = ""
		m.filterVariables()

	case "s":
		if err := m.config.SaveEnvFile(); err != nil {
			m.message = fmt.Sprintf("Error saving: %v", err)
		} else {
			m.saved = true
			m.message = "Configuration saved successfully!"
			// Update original vars to reflect saved state
			m.originalVars = make([]EnvVar, len(m.config.Variables))
			copy(m.originalVars, m.config.Variables)
		}

	case "r":
		// Reset to original values
		m.config.Variables = make([]EnvVar, len(m.originalVars))
		copy(m.config.Variables, m.originalVars)
		m.filteredVars = m.config.Variables
		m.message = "Changes reverted to last saved state"

	case "t":
		// Toggle secret visibility
		m.showSecrets = !m.showSecrets
		if m.showSecrets {
			m.message = "Showing secret values"
		} else {
			m.message = "Hiding secret values"
		}

	case "?":
		m.message = "Help: ↑/↓=navigate, Enter=edit, /=search, s=save, r=revert, t=toggle secrets, q=quit"
	}

	return m, nil
}

// handleEditMode handles key presses when editing a value
func (m *ConfigEditorModel) handleEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Save the edited value
		m.config.UpdateVariable(m.editingKey, m.editingValue)
		m.filterVariables() // Refresh filtered vars
		m.editMode = false
		m.message = fmt.Sprintf("Updated %s", m.editingKey)

	case "esc":
		// Cancel editing
		m.editMode = false
		m.editingValue = ""
		m.editingKey = ""

	case "backspace":
		if len(m.editingValue) > 0 {
			m.editingValue = m.editingValue[:len(m.editingValue)-1]
		}

	case "ctrl+u":
		// Clear the line
		m.editingValue = ""

	default:
		// Add character to editing value
		if len(msg.String()) == 1 {
			m.editingValue += msg.String()
		}
	}

	return m, nil
}

// handleSearchMode handles key presses when searching
func (m *ConfigEditorModel) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.searchMode = false

	case "backspace":
		if len(m.searchTerm) > 0 {
			m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
			m.filterVariables()
		}

	case "ctrl+u":
		m.searchTerm = ""
		m.filterVariables()

	default:
		if len(msg.String()) == 1 {
			m.searchTerm += msg.String()
			m.filterVariables()
		}
	}

	return m, nil
}

// filterVariables filters variables based on search term
func (m *ConfigEditorModel) filterVariables() {
	if m.searchTerm == "" {
		m.filteredVars = m.config.Variables
	} else {
		m.filteredVars = []EnvVar{}
		searchLower := strings.ToLower(m.searchTerm)
		
		for _, envVar := range m.config.Variables {
			if strings.Contains(strings.ToLower(envVar.Key), searchLower) ||
				strings.Contains(strings.ToLower(envVar.Value), searchLower) ||
				strings.Contains(strings.ToLower(envVar.Comment), searchLower) ||
				strings.Contains(strings.ToLower(envVar.Section), searchLower) {
				m.filteredVars = append(m.filteredVars, envVar)
			}
		}
	}
	
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.filteredVars) {
		m.cursor = max(0, len(m.filteredVars)-1)
	}
}

// View renders the configuration editor
func (m *ConfigEditorModel) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("DDALAB Configuration Editor")
	b.WriteString(title + "\n")
	
	// File path
	b.WriteString(fmt.Sprintf("File: %s\n\n", m.config.FilePath))

	// Search bar
	if m.searchMode {
		searchPrompt := inputStyle.Render(fmt.Sprintf("Search: %s█", m.searchTerm))
		b.WriteString(searchPrompt + "\n\n")
	} else if m.searchTerm != "" {
		searchInfo := fmt.Sprintf("Filter: '%s' (%d/%d vars)", m.searchTerm, len(m.filteredVars), len(m.config.Variables))
		b.WriteString(warningStyle.Render(searchInfo) + "\n\n")
	}

	// Edit mode
	if m.editMode {
		editPrompt := inputStyle.Render(fmt.Sprintf("Editing %s: %s█", m.editingKey, m.editingValue))
		b.WriteString(editPrompt + "\n\n")
	}

	// Table header
	header := fmt.Sprintf("%-30s %-40s %-20s %s", "KEY", "VALUE", "SECTION", "STATUS")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Variables table
	displayHeight := m.height - 15 // Account for header, title, etc.
	startIdx := max(0, m.cursor-displayHeight/2)
	endIdx := min(len(m.filteredVars), startIdx+displayHeight)

	var currentSection string
	for i := startIdx; i < endIdx; i++ {
		envVar := m.filteredVars[i]
		
		// Show section headers
		if envVar.Section != currentSection && envVar.Section != "" {
			currentSection = envVar.Section
			sectionHeader := sectionStyle.Render(fmt.Sprintf("── %s ──", currentSection))
			b.WriteString(sectionHeader + "\n")
		}

		// Format value display
		value := envVar.Value
		if envVar.IsSecret && !m.showSecrets && value != "" {
			value = strings.Repeat("*", min(len(value), 20))
		}
		if len(value) > 35 {
			value = value[:32] + "..."
		}

		// Format status
		status := ""
		if envVar.IsRequired {
			status += "REQ "
		}
		if envVar.IsSecret {
			status += "SEC "
		}
		if m.hasChanged(envVar) {
			status += "MOD"
		}

		// Format row
		row := fmt.Sprintf("%-30s %-40s %-20s %s", 
			truncate(envVar.Key, 28),
			truncate(value, 38),
			truncate(envVar.Section, 18),
			status,
		)

		// Apply styling
		var style lipgloss.Style
		if i == m.cursor {
			style = selectedStyle
		} else if envVar.IsRequired {
			style = requiredStyle
		} else if envVar.IsSecret {
			style = secretStyle
		} else {
			style = normalStyle
		}

		b.WriteString(style.Render(row) + "\n")
	}

	// Show scrolling indicator
	if len(m.filteredVars) > displayHeight {
		scrollInfo := fmt.Sprintf("(%d-%d of %d)", startIdx+1, endIdx, len(m.filteredVars))
		b.WriteString("\n" + helpStyle.Render(scrollInfo))
	}

	// Status message
	if m.message != "" {
		b.WriteString("\n" + warningStyle.Render(m.message))
	}

	// Help text
	if !m.editMode && !m.searchMode {
		help := "↑/↓: navigate • Enter: edit • /: search • s: save • r: revert • t: toggle secrets • q: quit"
		b.WriteString("\n" + helpStyle.Render(help))
	} else if m.editMode {
		help := "Enter: save • Esc: cancel • Ctrl+U: clear"
		b.WriteString("\n" + helpStyle.Render(help))
	} else if m.searchMode {
		help := "Type to search • Enter/Esc: exit search • Ctrl+U: clear"
		b.WriteString("\n" + helpStyle.Render(help))
	}

	return b.String()
}

// hasChanged checks if a variable has been modified
func (m *ConfigEditorModel) hasChanged(envVar EnvVar) bool {
	for _, original := range m.originalVars {
		if original.Key == envVar.Key {
			return original.Value != envVar.Value
		}
	}
	return false // New variable
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// RunConfigEditor runs the configuration editor
func RunConfigEditor(configPath string) error {
	// Load configuration
	config, err := LoadEnvFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create model
	model := NewConfigEditor(config)

	// Create program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run program
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run config editor: %w", err)
	}

	return nil
}