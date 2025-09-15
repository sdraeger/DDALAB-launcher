package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Common styles for consistent UI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(1, 2)

	menuHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("57")).
				Foreground(lipgloss.Color("230")).
				Padding(0, 1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

// MenuModel represents a selection menu
type MenuModel struct {
	title     string
	items     []string
	cursor    int
	selected  int
	choice    string
	cancelled bool
	width     int
	height    int
}

// NewMenuModel creates a new menu model
func NewMenuModel(title string, items []string) *MenuModel {
	return &MenuModel{
		title:    title,
		items:    items,
		selected: -1,
		width:    80,
		height:   20,
	}
}

func (m *MenuModel) Init() tea.Cmd {
	return nil
}

func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				// Wrap to last item when at the top
				m.cursor = len(m.items) - 1
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			} else {
				// Wrap to first item when at the bottom
				m.cursor = 0
			}

		case "enter", " ":
			m.selected = m.cursor
			m.choice = m.items[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *MenuModel) View() string {
	var b strings.Builder

	// Title
	if m.title != "" {
		b.WriteString(titleStyle.Render(m.title) + "\n\n")
	}

	// Menu items
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, item)

		if m.cursor == i {
			line = selectedItemStyle.Render(line)
		} else {
			line = normalItemStyle.Render(line)
		}

		b.WriteString(line + "\n")
	}

	// Help text
	b.WriteString("\n" + helpStyle.Render("↑/↓: navigate • Enter: select • q: quit"))

	return b.String()
}

// PromptModel represents a text input prompt
type PromptModel struct {
	title       string
	placeholder string
	value       string
	validate    func(string) error
	cancelled   bool
	errorMsg    string
	cursorPos   int
	width       int
	height      int
}

// NewPromptModel creates a new prompt model
func NewPromptModel(title, placeholder string, validate func(string) error) *PromptModel {
	return &PromptModel{
		title:       title,
		placeholder: placeholder,
		validate:    validate,
		width:       80,
		height:      10,
	}
}

func (m *PromptModel) Init() tea.Cmd {
	return nil
}

func (m *PromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if m.validate != nil {
				if err := m.validate(m.value); err != nil {
					m.errorMsg = err.Error()
					return m, nil
				}
			}
			return m, tea.Quit

		case "backspace":
			if len(m.value) > 0 && m.cursorPos > 0 {
				m.value = m.value[:m.cursorPos-1] + m.value[m.cursorPos:]
				m.cursorPos--
			}
			m.errorMsg = ""

		case "left":
			if m.cursorPos > 0 {
				m.cursorPos--
			}

		case "right":
			if m.cursorPos < len(m.value) {
				m.cursorPos++
			}

		case "home":
			m.cursorPos = 0

		case "end":
			m.cursorPos = len(m.value)

		case "ctrl+u":
			m.value = ""
			m.cursorPos = 0
			m.errorMsg = ""

		default:
			// Handle character input
			if len(msg.String()) == 1 && msg.String() >= " " {
				m.value = m.value[:m.cursorPos] + msg.String() + m.value[m.cursorPos:]
				m.cursorPos++
				m.errorMsg = ""
			}
		}
	}

	return m, nil
}

func (m *PromptModel) View() string {
	var b strings.Builder

	// Title
	if m.title != "" {
		b.WriteString(menuHeaderStyle.Render(m.title) + "\n\n")
	}

	// Input field
	displayValue := m.value
	if displayValue == "" && m.placeholder != "" {
		displayValue = m.placeholder
	}

	// Add cursor
	if m.cursorPos <= len(displayValue) {
		displayValue = displayValue[:m.cursorPos] + "█" + displayValue[m.cursorPos:]
	}

	inputField := promptStyle.Render(displayValue)
	b.WriteString(inputField + "\n")

	// Error message
	if m.errorMsg != "" {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.errorMsg) + "\n")
	}

	// Help text
	b.WriteString("\n" + helpStyle.Render("Enter: confirm • Ctrl+U: clear • Esc: cancel"))

	return b.String()
}

// ConfirmModel represents a yes/no confirmation dialog
type ConfirmModel struct {
	message   string
	choice    bool
	cancelled bool
	cursor    int
	width     int
	height    int
}

// NewConfirmModel creates a new confirmation model
func NewConfirmModel(message string) *ConfirmModel {
	return &ConfirmModel{
		message: message,
		width:   80,
		height:  10,
	}
}

func (m *ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m *ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "n":
			m.cancelled = true
			return m, tea.Quit

		case "left", "h":
			m.cursor = 0

		case "right", "l":
			m.cursor = 1

		case "y":
			m.choice = true
			return m, tea.Quit

		case "enter", " ":
			m.choice = m.cursor == 0
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *ConfirmModel) View() string {
	var b strings.Builder

	// Message
	b.WriteString(menuHeaderStyle.Render(m.message) + "\n\n")

	// Options
	options := []string{"Yes", "No"}
	for i, option := range options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, option)

		if m.cursor == i {
			line = selectedItemStyle.Render(line)
		} else {
			line = normalItemStyle.Render(line)
		}

		b.WriteString(line + "  ")
	}

	// Help text
	b.WriteString("\n\n" + helpStyle.Render("←/→: navigate • Enter/Space: select • y/n: quick select • Esc: cancel"))

	return b.String()
}

// WaitModel represents a simple "press enter to continue" prompt
type WaitModel struct {
	message   string
	completed bool
	width     int
	height    int
}

// NewWaitModel creates a new wait model
func NewWaitModel(message string) *WaitModel {
	if message == "" {
		message = "Press Enter to continue..."
	}
	return &WaitModel{
		message: message,
		width:   80,
		height:  5,
	}
}

func (m *WaitModel) Init() tea.Cmd {
	return nil
}

func (m *WaitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "ctrl+c", "esc", "q":
			m.completed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *WaitModel) View() string {
	return menuHeaderStyle.Render(m.message)
}

// UI Helper functions to run these models

// RunMenu displays a menu and returns the selected choice
func RunMenu(title string, items []string) (string, error) {
	model := NewMenuModel(title, items)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	menuModel := finalModel.(*MenuModel)
	if menuModel.cancelled {
		return "", fmt.Errorf("cancelled")
	}

	return menuModel.choice, nil
}

// RunPrompt displays a text input prompt and returns the entered value
func RunPrompt(title, placeholder string, validate func(string) error) (string, error) {
	model := NewPromptModel(title, placeholder, validate)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	promptModel := finalModel.(*PromptModel)
	if promptModel.cancelled {
		return "", fmt.Errorf("cancelled")
	}

	return promptModel.value, nil
}

// RunConfirm displays a yes/no confirmation and returns the choice
func RunConfirm(message string) (bool, error) {
	model := NewConfirmModel(message)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	confirmModel := finalModel.(*ConfirmModel)
	if confirmModel.cancelled {
		return false, nil
	}

	return confirmModel.choice, nil
}

// RunWait displays a "press enter to continue" message
func RunWait(message string) error {
	model := NewWaitModel(message)
	p := tea.NewProgram(model)

	_, err := p.Run()
	return err
}
