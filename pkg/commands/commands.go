package commands

import (
	"context"
	"fmt"

	"github.com/ddalab/launcher/pkg/api"
	"github.com/ddalab/launcher/pkg/config"
)

// Commander handles DDALAB operations via API
type Commander struct {
	configManager *config.ConfigManager
	apiClient     *api.Client
}

// NewCommander creates a new commander instance that uses the API client
func NewCommander(configManager *config.ConfigManager, apiClient *api.Client) *Commander {
	return &Commander{
		configManager: configManager,
		apiClient:     apiClient,
	}
}

// Start starts the DDALAB services
func (c *Commander) Start() error {
	return c.StartWithContext(context.Background())
}

// StartWithContext starts the DDALAB services with cancellation support via API
func (c *Commander) StartWithContext(ctx context.Context) error {
	err := c.apiClient.StartStack(ctx)
	if err != nil {
		return fmt.Errorf("failed to start DDALAB: %w", err)
	}

	c.configManager.SetLastOperation("start")
	_ = c.configManager.Save()

	return nil
}

// Stop stops the DDALAB services via API
func (c *Commander) Stop() error {
	ctx := context.Background()
	err := c.apiClient.StopStack(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop DDALAB: %w", err)
	}

	c.configManager.SetLastOperation("stop")
	_ = c.configManager.Save()

	return nil
}

// Restart restarts the DDALAB services via API
func (c *Commander) Restart() error {
	ctx := context.Background()
	err := c.apiClient.RestartStack(ctx)
	if err != nil {
		return fmt.Errorf("failed to restart DDALAB: %w", err)
	}

	c.configManager.SetLastOperation("restart")
	_ = c.configManager.Save()

	return nil
}

// Status checks the status of DDALAB services via API
func (c *Commander) Status() (string, error) {
	ctx := context.Background()
	status, err := c.apiClient.GetStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get DDALAB status: %w", err)
	}

	// Format the status for display
	result := fmt.Sprintf("DDALAB Status: %s\n", status.State)
	result += fmt.Sprintf("Running: %t\n", status.Running)
	result += fmt.Sprintf("Installation Path: %s\n", status.Installation.Path)

	if len(status.Services) > 0 {
		result += "\nServices:\n"
		for _, service := range status.Services {
			result += fmt.Sprintf("  %s: %s (%s)\n", service.Name, service.Status, service.Health)
		}
	}

	return result, nil
}

// Logs retrieves DDALAB service logs
func (c *Commander) Logs() (string, error) {
	return c.LogsWithContext(context.Background())
}

// LogsWithContext retrieves DDALAB service logs with cancellation support via API
func (c *Commander) LogsWithContext(ctx context.Context) (string, error) {
	logs, err := c.apiClient.GetLogs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get DDALAB logs: %w", err)
	}

	return logs, nil
}

// Backup creates a database backup via API
func (c *Commander) Backup() error {
	ctx := context.Background()
	filename, err := c.apiClient.CreateBackup(ctx)
	if err != nil {
		return fmt.Errorf("failed to backup DDALAB: %w", err)
	}

	fmt.Printf("Backup created: %s\n", filename)

	c.configManager.SetLastOperation("backup")
	_ = c.configManager.Save()

	return nil
}

// Update updates DDALAB to the latest version
func (c *Commander) Update() error {
	return c.UpdateWithContext(context.Background())
}

// UpdateWithContext updates DDALAB to the latest version with cancellation support via API
func (c *Commander) UpdateWithContext(ctx context.Context) error {
	err := c.apiClient.UpdateStack(ctx)
	if err != nil {
		return fmt.Errorf("failed to update DDALAB: %w", err)
	}

	c.configManager.SetLastOperation("update")
	_ = c.configManager.Save()

	return nil
}

// Uninstall removes DDALAB (stops services and removes volumes) via API
func (c *Commander) Uninstall() error {
	ctx := context.Background()

	// Stop services first
	err := c.apiClient.StopStack(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop DDALAB services: %w", err)
	}

	// Note: Full uninstall functionality would need to be implemented in the backend
	// For now, we just stop the services
	fmt.Println("DDALAB services stopped. Complete uninstall functionality requires backend implementation.")

	c.configManager.SetLastOperation("uninstall")
	_ = c.configManager.Save()

	return nil
}

// IsRunning checks if DDALAB services are currently running via API
func (c *Commander) IsRunning() (bool, error) {
	ctx := context.Background()
	status, err := c.apiClient.GetStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check service status: %w", err)
	}

	return status.Running, nil
}

// GetServiceHealth returns health information about DDALAB services via API
func (c *Commander) GetServiceHealth() (map[string]string, error) {
	ctx := context.Background()
	status, err := c.apiClient.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get service health: %w", err)
	}

	// Convert to map format for UI display
	services := make(map[string]string)
	for _, service := range status.Services {
		serviceStatus := service.Status
		if service.Health != "" && service.Health != service.Status {
			serviceStatus += " (" + service.Health + ")"
		}
		if service.Uptime != "" {
			serviceStatus += " " + service.Uptime
		}
		services[service.Name] = serviceStatus
	}

	return services, nil
}
