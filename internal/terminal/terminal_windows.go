//go:build windows
// +build windows

package terminal

import (
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
)

// isTerminalPlatform checks if running in a terminal on Windows
func isTerminalPlatform() bool {
	// Check if we have a console window
	ret, _, _ := procGetConsoleWindow.Call()
	if ret == 0 {
		return false
	}

	// Also check if stdin is a console
	handle, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return false
	}

	var mode uint32
	ret, _, _ = procGetConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	return ret != 0
}
