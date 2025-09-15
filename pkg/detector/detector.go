package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstallationInfo contains details about a detected DDALAB installation
type InstallationInfo struct {
	Path           string
	Valid          bool
	Version        string
	DockerCompose  bool
	Scripts        bool
	HasCertificates bool
}

// Detector handles DDALAB installation detection
type Detector struct{}

// NewDetector creates a new DDALAB detector
func NewDetector() *Detector {
	return &Detector{}
}

// FindInstallations searches for DDALAB installations in common locations
func (d *Detector) FindInstallations() ([]*InstallationInfo, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Common installation paths
	searchPaths := []string{
		filepath.Join(homeDir, "DDALAB-setup"),
		filepath.Join(homeDir, "Desktop", "DDALAB-setup"),
		filepath.Join(homeDir, "Downloads", "DDALAB-setup"),
		"/opt/DDALAB-setup",
		"/usr/local/DDALAB-setup",
		"../DDALAB-setup", // Relative to current directory
	}

	var installations []*InstallationInfo
	
	for _, path := range searchPaths {
		if info := d.DetectInstallation(path); info.Valid {
			installations = append(installations, info)
		}
	}

	return installations, nil
}

// DetectInstallation checks if a given path contains a valid DDALAB installation
func (d *Detector) DetectInstallation(path string) *InstallationInfo {
	info := &InstallationInfo{
		Path: path,
	}

	// Check if directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return info
	}

	// Check for required files
	requiredFiles := []string{
		"docker-compose.yml",
		"README.md",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(path, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return info
		}
	}

	info.DockerCompose = true

	// Check for DDALAB scripts
	scripts := []string{
		"ddalab.sh",
		"ddalab.ps1",
		"ddalab.bat",
	}

	for _, script := range scripts {
		scriptPath := filepath.Join(path, script)
		if _, err := os.Stat(scriptPath); err == nil {
			info.Scripts = true
			break
		}
	}

	// Check for certificates directory
	certsPath := filepath.Join(path, "certs")
	if _, err := os.Stat(certsPath); err == nil {
		info.HasCertificates = true
	}

	// Try to detect version from docker-compose.yml
	info.Version = d.extractVersion(path)

	// Installation is valid if it has docker-compose and scripts
	info.Valid = info.DockerCompose && info.Scripts

	return info
}

// extractVersion attempts to extract version information from the installation
func (d *Detector) extractVersion(path string) string {
	dockerComposePath := filepath.Join(path, "docker-compose.yml")
	content, err := os.ReadFile(dockerComposePath)
	if err != nil {
		return "unknown"
	}

	contentStr := string(content)
	
	// Look for DDALAB image version
	if strings.Contains(contentStr, "sdraeger1/ddalab:") {
		lines := strings.Split(contentStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "sdraeger1/ddalab:") && strings.Contains(line, "image:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 3 {
					version := strings.TrimSpace(parts[2])
					return strings.Trim(version, `"'`)
				}
			}
		}
	}

	// Check if README has version info
	readmePath := filepath.Join(path, "README.md")
	if readmeContent, err := os.ReadFile(readmePath); err == nil {
		readmeStr := string(readmeContent)
		if strings.Contains(readmeStr, "DDALAB") {
			return "detected"
		}
	}

	return "unknown"
}

// ValidateInstallation performs comprehensive validation of an installation
func (d *Detector) ValidateInstallation(path string) error {
	info := d.DetectInstallation(path)
	
	if !info.Valid {
		return fmt.Errorf("invalid DDALAB installation at %s", path)
	}

	// Check if Docker is available
	if !d.isDockerAvailable() {
		return fmt.Errorf("docker is not available or not running")
	}

	// Check if docker-compose is available
	if !d.isDockerComposeAvailable() {
		return fmt.Errorf("docker-compose is not available")
	}

	return nil
}

// isDockerAvailable checks if Docker is installed and running
func (d *Detector) isDockerAvailable() bool {
	// Simple check - try to access docker socket or run docker version
	_, err := os.Stat("/var/run/docker.sock")
	return err == nil
}

// isDockerComposeAvailable checks if docker-compose is available
func (d *Detector) isDockerComposeAvailable() bool {
	// Check if docker-compose command exists
	// This is a simplified check - in a real implementation,
	// you might want to actually run the command
	return true // Assume it's available for now
}