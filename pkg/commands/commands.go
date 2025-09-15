package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ddalab/launcher/pkg/config"
)

// Commander handles DDALAB operations
type Commander struct {
	configManager *config.ConfigManager
}

// NewCommander creates a new commander instance
func NewCommander(configManager *config.ConfigManager) *Commander {
	return &Commander{
		configManager: configManager,
	}
}

// Start starts the DDALAB services
func (c *Commander) Start() error {
	return c.StartWithContext(context.Background())
}

// StartWithContext starts the DDALAB services with cancellation support
func (c *Commander) StartWithContext(ctx context.Context) error {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	script := c.getScriptName()
	scriptPath := filepath.Join(ddalabPath, script)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("DDALAB script not found at %s", scriptPath)
	}

	cmd := c.createCommandWithContext(ctx, scriptPath, "start")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to start DDALAB: %s\nOutput: %s", err, string(output))
	}

	c.configManager.SetLastOperation("start")
	c.configManager.Save()
	
	return nil
}

// Stop stops the DDALAB services
func (c *Commander) Stop() error {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	script := c.getScriptName()
	scriptPath := filepath.Join(ddalabPath, script)

	cmd := c.createCommand(scriptPath, "stop")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop DDALAB: %s\nOutput: %s", err, string(output))
	}

	c.configManager.SetLastOperation("stop")
	c.configManager.Save()
	
	return nil
}

// Restart restarts the DDALAB services
func (c *Commander) Restart() error {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	script := c.getScriptName()
	scriptPath := filepath.Join(ddalabPath, script)

	cmd := c.createCommand(scriptPath, "restart")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart DDALAB: %s\nOutput: %s", err, string(output))
	}

	c.configManager.SetLastOperation("restart")
	c.configManager.Save()
	
	return nil
}

// Status checks the status of DDALAB services
func (c *Commander) Status() (string, error) {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return "", fmt.Errorf("DDALAB path not configured")
	}

	script := c.getScriptName()
	scriptPath := filepath.Join(ddalabPath, script)

	cmd := c.createCommand(scriptPath, "status")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get DDALAB status: %s", err)
	}

	return string(output), nil
}

// Logs retrieves DDALAB service logs
func (c *Commander) Logs() (string, error) {
	return c.LogsWithContext(context.Background())
}

// LogsWithContext retrieves DDALAB service logs with cancellation support
func (c *Commander) LogsWithContext(ctx context.Context) (string, error) {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return "", fmt.Errorf("DDALAB path not configured")
	}

	script := c.getScriptName()
	scriptPath := filepath.Join(ddalabPath, script)

	cmd := c.createCommandWithContext(ctx, scriptPath, "logs")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("failed to get DDALAB logs: %s", err)
	}

	return string(output), nil
}

// Backup creates a database backup
func (c *Commander) Backup() error {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	script := c.getScriptName()
	scriptPath := filepath.Join(ddalabPath, script)

	cmd := c.createCommand(scriptPath, "backup")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to backup DDALAB: %s\nOutput: %s", err, string(output))
	}

	c.configManager.SetLastOperation("backup")
	c.configManager.Save()
	
	return nil
}

// Update updates DDALAB to the latest version
func (c *Commander) Update() error {
	return c.UpdateWithContext(context.Background())
}

// UpdateWithContext updates DDALAB to the latest version with cancellation support
func (c *Commander) UpdateWithContext(ctx context.Context) error {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	// Stop services first
	if err := c.Stop(); err != nil {
		return fmt.Errorf("failed to stop services before update: %w", err)
	}

	// Pull latest images with context
	cmd := exec.CommandContext(ctx, "docker-compose", "pull")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to pull latest images: %s\nOutput: %s", err, string(output))
	}

	// Start services with new images
	if err := c.StartWithContext(ctx); err != nil {
		return fmt.Errorf("failed to start services after update: %w", err)
	}

	c.configManager.SetLastOperation("update")
	c.configManager.Save()
	
	return nil
}

// Uninstall removes DDALAB (stops services and removes volumes)
func (c *Commander) Uninstall() error {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	// Stop and remove all containers and volumes
	cmd := exec.Command("docker-compose", "down", "-v", "--remove-orphans")
	cmd.Dir = ddalabPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove DDALAB containers: %s\nOutput: %s", err, string(output))
	}

	// Remove any DDALAB images
	cmd = exec.Command("docker", "image", "prune", "-a", "-f", "--filter", "label=com.ddalab.component")
	output, err = cmd.CombinedOutput()
	if err != nil {
		// Non-critical error, continue
		fmt.Printf("Warning: failed to remove DDALAB images: %s\n", err)
	}

	c.configManager.SetLastOperation("uninstall")
	c.configManager.Save()
	
	return nil
}

// getScriptName returns the appropriate script name for the current OS
func (c *Commander) getScriptName() string {
	switch runtime.GOOS {
	case "windows":
		// Check for .ps1 first, then .bat
		ddalabPath := c.configManager.GetDDALABPath()
		if _, err := os.Stat(filepath.Join(ddalabPath, "ddalab.ps1")); err == nil {
			return "ddalab.ps1"
		}
		return "ddalab.bat"
	default:
		return "ddalab.sh"
	}
}

// createCommand creates an appropriate command for the current OS
func (c *Commander) createCommand(scriptPath, action string) *exec.Cmd {
	return c.createCommandWithContext(context.Background(), scriptPath, action)
}

// createCommandWithContext creates an appropriate command with context for the current OS
func (c *Commander) createCommandWithContext(ctx context.Context, scriptPath, action string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		if strings.HasSuffix(scriptPath, ".ps1") {
			return exec.CommandContext(ctx, "powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath, action)
		}
		return exec.CommandContext(ctx, scriptPath, action)
	default:
		return exec.CommandContext(ctx, "bash", scriptPath, action)
	}
}

// IsRunning checks if DDALAB services are currently running
func (c *Commander) IsRunning() (bool, error) {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return false, fmt.Errorf("DDALAB path not configured")
	}

	// Use docker-compose ps to check running services
	cmd := exec.Command("docker-compose", "ps", "-q")
	cmd.Dir = ddalabPath
	
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check service status: %w", err)
	}

	// If there are running containers, output will not be empty
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetServiceHealth returns health information about DDALAB services
func (c *Commander) GetServiceHealth() (map[string]string, error) {
	ddalabPath := c.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return nil, fmt.Errorf("DDALAB path not configured")
	}

	cmd := exec.Command("docker-compose", "ps", "--format", "table")
	cmd.Dir = ddalabPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get service health: %w", err)
	}

	// Parse the output to extract service health
	// This is a simplified implementation
	services := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "ddalab") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				serviceName := fields[0]
				status := "unknown"
				if len(fields) >= 4 {
					status = fields[3]
				}
				services[serviceName] = status
			}
		}
	}

	return services, nil
}