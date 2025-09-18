package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/ddalab/launcher/pkg/api"
	"github.com/ddalab/launcher/pkg/mode"
)

// Dispatcher routes commands to either API or local implementations
type Dispatcher struct {
	modeManager *mode.Manager
	commander   *Commander // existing local commander
}

// NewDispatcher creates a new command dispatcher
func NewDispatcher(modeManager *mode.Manager, commander *Commander) *Dispatcher {
	return &Dispatcher{
		modeManager: modeManager,
		commander:   commander,
	}
}

// ExecuteCommand executes a command using API mode with bootstrap fallback
func (d *Dispatcher) ExecuteCommand(command string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return d.ExecuteCommandWithContext(ctx, command, args...)
}

// ExecuteCommandWithContext executes a command with a provided context
func (d *Dispatcher) ExecuteCommandWithContext(ctx context.Context, command string, args ...string) error {
	// Always try API mode first
	if d.modeManager.IsAPIMode() {
		return d.executeAPICommand(ctx, command, args...)
	}

	// If not in API mode, try to bootstrap and switch to API mode
	if d.modeManager.GetBootstrapper().CanBootstrap() {
		if err := d.modeManager.PerformBootstrap(); err == nil {
			// Bootstrap succeeded, now execute via API
			return d.executeAPICommand(ctx, command, args...)
		}
	}

	// If bootstrap fails, return appropriate error
	return fmt.Errorf("API mode unavailable and bootstrap failed - ensure Docker is running")
}

// executeAPICommand executes commands via the Docker extension API
func (d *Dispatcher) executeAPICommand(ctx context.Context, command string, args ...string) error {
	apiClient := d.modeManager.GetAPIClient()
	if apiClient == nil {
		return fmt.Errorf("API client not available in non-API mode")
	}

	switch command {
	case "start":
		return apiClient.StartStack(ctx)
	case "stop":
		return apiClient.StopStack(ctx)
	case "restart":
		return apiClient.RestartStack(ctx)
	case "backup":
		filename, err := apiClient.CreateBackup(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Backup created: %s\n", filename)
		return nil
	case "update":
		return apiClient.UpdateDDALAB(ctx)
	case "logs":
		logs, err := apiClient.GetLogs(ctx)
		if err != nil {
			return err
		}
		fmt.Println(logs)
		return nil
	case "status":
		status, err := apiClient.GetStatus(ctx)
		if err != nil {
			return err
		}
		d.printAPIStatus(status)
		return nil
	default:
		return fmt.Errorf("command '%s' not supported in API mode", command)
	}
}

// GetStatus returns status information using API mode with bootstrap fallback
func (d *Dispatcher) GetStatus() (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Always try API mode first
	if d.modeManager.IsAPIMode() {
		apiClient := d.modeManager.GetAPIClient()
		if apiClient == nil {
			return nil, fmt.Errorf("API client not available")
		}
		return apiClient.GetStatus(ctx)
	}

	// If not in API mode, try to bootstrap and switch to API mode
	if d.modeManager.GetBootstrapper().CanBootstrap() {
		if err := d.modeManager.PerformBootstrap(); err == nil {
			// Bootstrap succeeded, now get status via API
			apiClient := d.modeManager.GetAPIClient()
			if apiClient != nil {
				return apiClient.GetStatus(ctx)
			}
		}
	}

	// If bootstrap fails, return error
	return nil, fmt.Errorf("API mode unavailable and bootstrap failed - ensure Docker is running")
}

// printAPIStatus prints status information from the API
func (d *Dispatcher) printAPIStatus(status *api.Status) {
	fmt.Printf("DDALAB Status: %s\n", getStatusText(status.Running))
	fmt.Printf("Version: %s\n", status.Installation.Version)
	fmt.Printf("Path: %s\n", status.Installation.Path)
	fmt.Println("\nServices:")

	for _, service := range status.Services {
		statusIcon := "❌"
		if service.Status == "running" {
			statusIcon = "✅"
		} else if service.Status == "starting" {
			statusIcon = "🔄"
		}
		fmt.Printf("  %s %s: %s\n", statusIcon, service.Name, service.Status)
	}
}

// getStatusText converts boolean status to readable text
func getStatusText(running bool) string {
	if running {
		return "Running ✅"
	}
	return "Stopped ❌"
}

// IsAPIMode returns true if currently in API mode
func (d *Dispatcher) IsAPIMode() bool {
	return d.modeManager.IsAPIMode()
}

// IsLocalMode returns true if currently in local mode
func (d *Dispatcher) IsLocalMode() bool {
	return d.modeManager.IsLocalMode()
}

// GetModeDescription returns a description of the current mode
func (d *Dispatcher) GetModeDescription() string {
	return d.modeManager.GetModeDescription()
}
