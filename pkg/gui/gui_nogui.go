//go:build nogui

package gui

import (
	"fmt"

	"github.com/ddalab/launcher/pkg/commands"
	"github.com/ddalab/launcher/pkg/config"
	"github.com/ddalab/launcher/pkg/status"
)

// GUI is a stub implementation when GUI is disabled
type GUI struct{}

// NewGUI creates a stub GUI instance when GUI is disabled
func NewGUI(commander *commands.Commander, configMgr *config.ConfigManager, statusMonitor *status.Monitor) *GUI {
	return &GUI{}
}

// Show displays an error message when GUI is disabled
func (g *GUI) Show() {
	fmt.Println("❌ GUI functionality is not available in this build.")
	fmt.Println("This launcher was built without GUI support.")
	fmt.Println("To enable GUI, rebuild with: go build -tags gui")
}
