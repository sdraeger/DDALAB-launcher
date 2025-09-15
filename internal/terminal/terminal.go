package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// IsTerminal checks if the program is running in a terminal
func IsTerminal() bool {
	return isTerminalPlatform()
}

// RelaunchInTerminal attempts to relaunch the program in a terminal
func RelaunchInTerminal() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return relaunchInMacTerminal(executable)
	case "linux":
		return relaunchInLinuxTerminal(executable)
	case "windows":
		return relaunchInWindowsTerminal(executable)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// relaunchInMacTerminal relaunches in Terminal.app on macOS
func relaunchInMacTerminal(executable string) error {
	// AppleScript to open Terminal and run our program
	script := fmt.Sprintf(`
		tell application "Terminal"
			activate
			do script "%s; exit"
		end tell
	`, executable)
	
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Start()
}

// relaunchInLinuxTerminal tries various terminal emulators on Linux
func relaunchInLinuxTerminal(executable string) error {
	// Try common terminal emulators in order of preference
	terminals := []struct {
		name string
		args []string
	}{
		{"gnome-terminal", []string{"--", executable}},
		{"konsole", []string{"-e", executable}},
		{"xfce4-terminal", []string{"-e", executable}},
		{"mate-terminal", []string{"-e", executable}},
		{"xterm", []string{"-e", executable}},
		{"rxvt", []string{"-e", executable}},
		{"terminator", []string{"-e", executable}},
		{"alacritty", []string{"-e", executable}},
		{"kitty", []string{executable}},
	}

	for _, term := range terminals {
		if _, err := exec.LookPath(term.name); err == nil {
			cmd := exec.Command(term.name, term.args...)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable terminal emulator found")
}

// relaunchInWindowsTerminal relaunches in a Windows terminal
func relaunchInWindowsTerminal(executable string) error {
	// First try Windows Terminal (if available)
	if _, err := exec.LookPath("wt.exe"); err == nil {
		cmd := exec.Command("wt.exe", executable)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	// Fall back to cmd.exe with a new window
	cmd := exec.Command("cmd.exe", "/c", "start", "DDALAB Launcher", "/wait", executable)
	return cmd.Start()
}


// ShowGUIError displays an error message using a GUI dialog
func ShowGUIError(title, message string) {
	switch runtime.GOOS {
	case "darwin":
		showMacDialog(title, message)
	case "linux":
		showLinuxDialog(title, message)
	case "windows":
		showWindowsDialog(title, message)
	}
}

// showMacDialog shows a dialog on macOS using osascript
func showMacDialog(title, message string) {
	script := fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"OK"} default button "OK"`, 
		message, title)
	exec.Command("osascript", "-e", script).Run()
}

// showLinuxDialog shows a dialog on Linux using available tools
func showLinuxDialog(title, message string) {
	// Try different dialog tools
	tools := []struct {
		name string
		args []string
	}{
		{"zenity", []string{"--error", "--title=" + title, "--text=" + message}},
		{"kdialog", []string{"--error", message, "--title", title}},
		{"xmessage", []string{"-center", "-title", title, message}},
		{"notify-send", []string{"-u", "critical", title, message}},
	}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool.name); err == nil {
			exec.Command(tool.name, tool.args...).Run()
			return
		}
	}
}

// showWindowsDialog shows a dialog on Windows
func showWindowsDialog(title, message string) {
	// Use PowerShell to show a message box
	script := fmt.Sprintf(`[System.Windows.Forms.MessageBox]::Show('%s', '%s', 'OK', 'Error')`,
		message, title)
	exec.Command("powershell", "-Command", 
		"Add-Type -AssemblyName System.Windows.Forms;", script).Run()
}