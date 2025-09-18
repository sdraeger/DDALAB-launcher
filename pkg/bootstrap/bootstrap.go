package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Bootstrap provides minimal functionality to start the Docker extension backend
// when it's not available. This is a fallback mechanism for situations where
// the launcher needs to operate independently.
type Bootstrap struct {
	extensionPath string
	isAvailable   bool
}

// NewBootstrap creates a new bootstrap instance
func NewBootstrap() *Bootstrap {
	return &Bootstrap{}
}

// CheckDockerExtension checks if Docker Desktop and the DDALAB extension are available
func (b *Bootstrap) CheckDockerExtension() error {
	// First, check if Docker is running
	if err := b.checkDockerRunning(); err != nil {
		return fmt.Errorf("Docker is not running: %w", err)
	}

	// Check if Docker Desktop is installed (not just Docker Engine)
	if !b.isDockerDesktop() {
		return fmt.Errorf("Docker Desktop is required but not found")
	}

	// Try to find the DDALAB extension
	extensionPath, err := b.findExtension()
	if err != nil {
		return fmt.Errorf("DDALAB Docker extension not found: %w", err)
	}

	b.extensionPath = extensionPath
	b.isAvailable = true
	return nil
}

// checkDockerRunning verifies Docker daemon is accessible
func (b *Bootstrap) checkDockerRunning() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon not accessible")
	}

	return nil
}

// isDockerDesktop checks if Docker Desktop is installed (not just Docker Engine)
func (b *Bootstrap) isDockerDesktop() bool {
	// Check for Docker Desktop specific paths
	switch runtime.GOOS {
	case "darwin":
		// macOS: Check for Docker.app
		if _, err := os.Stat("/Applications/Docker.app"); err == nil {
			return true
		}
	case "windows":
		// Windows: Check for Docker Desktop executable
		if _, err := exec.LookPath("Docker Desktop.exe"); err == nil {
			return true
		}
	case "linux":
		// Linux: Check for docker-desktop
		if _, err := os.Stat("/usr/bin/docker-desktop"); err == nil {
			return true
		}
		// Also check systemd service
		cmd := exec.Command("systemctl", "is-active", "docker-desktop")
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	return false
}

// findExtension attempts to locate the DDALAB Docker extension
func (b *Bootstrap) findExtension() (string, error) {
	// Common paths where Docker extensions are installed
	var searchPaths []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		searchPaths = []string{
			filepath.Join(homeDir, "Library/Containers/com.docker.docker/Data/extensions"),
			filepath.Join(homeDir, ".docker/desktop/extensions"),
		}
	case "windows":
		searchPaths = []string{
			filepath.Join(os.Getenv("APPDATA"), "Docker/extensions"),
			filepath.Join(homeDir, ".docker/desktop/extensions"),
		}
	case "linux":
		searchPaths = []string{
			filepath.Join(homeDir, ".docker/desktop/extensions"),
			"/usr/local/share/docker/extensions",
		}
	}

	// Look for DDALAB extension in standard locations
	for _, basePath := range searchPaths {
		ddalabPath := filepath.Join(basePath, "ddalab")
		if _, err := os.Stat(ddalabPath); err == nil {
			return ddalabPath, nil
		}

		// Also check with full extension ID
		ddalabFullPath := filepath.Join(basePath, "simonmcnair/ddalab")
		if _, err := os.Stat(ddalabFullPath); err == nil {
			return ddalabFullPath, nil
		}
	}

	return "", fmt.Errorf("extension not found in standard locations")
}

// StartExtensionBackend attempts to start the Docker extension backend service
func (b *Bootstrap) StartExtensionBackend(ctx context.Context) error {
	if !b.isAvailable {
		return fmt.Errorf("Docker extension not available")
	}

	// Check if the extension backend is already running
	if b.isBackendRunning() {
		return nil
	}

	// Start the extension backend
	// This is a placeholder - actual implementation would depend on how
	// the Docker extension backend can be started independently
	return fmt.Errorf("manual extension backend start not implemented")
}

// isBackendRunning checks if the extension backend is responding
func (b *Bootstrap) isBackendRunning() bool {
	// Try to connect to the default API endpoint
	cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8080/api/v1/health")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return string(output) == "200"
}

// StartMinimalServices starts only the essential DDALAB services locally
// This is used when the Docker extension is not available
func (b *Bootstrap) StartMinimalServices(ctx context.Context, ddalabPath string) error {
	// Check if docker-compose.yml exists
	composeFile := filepath.Join(ddalabPath, "docker-compose.yml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found in %s", ddalabPath)
	}

	// Start only core services (postgres, redis, api)
	cmd := exec.CommandContext(ctx, "docker-compose",
		"-f", composeFile,
		"up", "-d",
		"postgres", "redis", "ddalab")

	cmd.Dir = ddalabPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start minimal services: %w", err)
	}

	return nil
}

// GetBootstrapMode returns the current bootstrap capability
func (b *Bootstrap) GetBootstrapMode() string {
	if b.isAvailable {
		return "Docker Extension Available"
	}
	if b.isDockerDesktop() {
		return "Docker Desktop (No Extension)"
	}
	if b.checkDockerRunning() == nil {
		return "Docker Engine Only"
	}
	return "No Docker"
}

// CanBootstrap returns true if some form of bootstrap is possible
func (b *Bootstrap) CanBootstrap() bool {
	return b.checkDockerRunning() == nil
}

// IsExtensionAvailable returns true if the Docker extension was found
func (b *Bootstrap) IsExtensionAvailable() bool {
	return b.isAvailable
}
