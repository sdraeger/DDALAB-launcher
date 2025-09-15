//go:build darwin || linux
// +build darwin linux

package terminal

import (
	"os"
)

// isTerminalPlatform checks if running in a terminal on Unix systems
func isTerminalPlatform() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}