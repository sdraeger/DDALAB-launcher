package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Version is injected at build time - this is a fallback for development
var Version = "dev"

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// SetVersion allows setting the version from the main package
func SetVersion(v string) {
	Version = v
}

// OperationMode defines how the launcher operates
type OperationMode string

const (
	// ModeLocal is deprecated - kept for config compatibility, now acts as auto mode
	ModeLocal OperationMode = "local"
	// ModeAPI uses the Docker extension API at port 8080
	ModeAPI OperationMode = "api"
	// ModeAuto automatically detects the best mode (API with bootstrap fallback)
	ModeAuto OperationMode = "auto"
)

// LauncherConfig holds the persistent state of the launcher
type LauncherConfig struct {
	DDALABPath          string        `json:"ddalab_path"`
	FirstRun            bool          `json:"first_run"`
	LastOperation       string        `json:"last_operation"`
	Version             string        `json:"version"`
	AutoUpdateCheck     bool          `json:"auto_update_check"`
	LastUpdateCheck     time.Time     `json:"last_update_check"`
	UpdateCheckInterval int           `json:"update_check_interval_hours"` // in hours
	OperationMode       OperationMode `json:"operation_mode"`              // mode: api or auto (local deprecated)
	APIEndpoint         string        `json:"api_endpoint"`                // Docker extension API endpoint
}

// ConfigManager handles loading and saving configuration
type ConfigManager struct {
	configPath string
	config     *LauncherConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".ddalab-launcher")

	cm := &ConfigManager{
		configPath: configPath,
		config: &LauncherConfig{
			FirstRun:            true,
			Version:             GetVersion(),
			AutoUpdateCheck:     true,                        // Default to enabled
			UpdateCheckInterval: 24,                          // Check daily by default
			LastUpdateCheck:     time.Time{},                 // Never checked
			OperationMode:       ModeAuto,                    // Default to auto-detection
			APIEndpoint:         "http://localhost:8080/api", // Docker extension API
		},
	}

	// Try to load existing config
	if err := cm.Load(); err != nil {
		// If config doesn't exist, that's OK for first run
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return cm, nil
}

// Load reads the configuration from disk
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, cm.config)
}

// Save writes the configuration to disk
func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configPath, data, 0644)
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *LauncherConfig {
	return cm.config
}

// SetDDALABPath sets the DDALAB installation path
func (cm *ConfigManager) SetDDALABPath(path string) {
	cm.config.DDALABPath = path
	cm.config.FirstRun = false
}

// SetLastOperation records the last operation performed
func (cm *ConfigManager) SetLastOperation(operation string) {
	cm.config.LastOperation = operation
}

// IsFirstRun returns true if this is the first time running the launcher
func (cm *ConfigManager) IsFirstRun() bool {
	return cm.config.FirstRun
}

// GetDDALABPath returns the configured DDALAB path
func (cm *ConfigManager) GetDDALABPath() string {
	return cm.config.DDALABPath
}

// Update-related methods

// SetAutoUpdateCheck enables or disables automatic update checking
func (cm *ConfigManager) SetAutoUpdateCheck(enabled bool) {
	cm.config.AutoUpdateCheck = enabled
}

// IsAutoUpdateCheckEnabled returns true if automatic update checking is enabled
func (cm *ConfigManager) IsAutoUpdateCheckEnabled() bool {
	return cm.config.AutoUpdateCheck
}

// SetUpdateCheckInterval sets the interval between update checks in hours
func (cm *ConfigManager) SetUpdateCheckInterval(hours int) {
	cm.config.UpdateCheckInterval = hours
}

// GetUpdateCheckInterval returns the update check interval in hours
func (cm *ConfigManager) GetUpdateCheckInterval() int {
	return cm.config.UpdateCheckInterval
}

// SetLastUpdateCheck records when we last checked for updates
func (cm *ConfigManager) SetLastUpdateCheck(t time.Time) {
	cm.config.LastUpdateCheck = t
}

// GetLastUpdateCheck returns when we last checked for updates
func (cm *ConfigManager) GetLastUpdateCheck() time.Time {
	return cm.config.LastUpdateCheck
}

// ShouldCheckForUpdates determines if we should check for updates now
func (cm *ConfigManager) ShouldCheckForUpdates() bool {
	if !cm.config.AutoUpdateCheck {
		return false
	}

	interval := time.Duration(cm.config.UpdateCheckInterval) * time.Hour
	return time.Since(cm.config.LastUpdateCheck) >= interval
}

// Operation mode related methods

// SetOperationMode sets the operation mode (killswitch)
func (cm *ConfigManager) SetOperationMode(mode OperationMode) {
	cm.config.OperationMode = mode
}

// GetOperationMode returns the current operation mode
func (cm *ConfigManager) GetOperationMode() OperationMode {
	return cm.config.OperationMode
}

// SetAPIEndpoint sets the API endpoint for Docker extension communication
func (cm *ConfigManager) SetAPIEndpoint(endpoint string) {
	cm.config.APIEndpoint = endpoint
}

// GetAPIEndpoint returns the API endpoint
func (cm *ConfigManager) GetAPIEndpoint() string {
	return cm.config.APIEndpoint
}

// IsAPIMode returns true if the launcher should use API mode
func (cm *ConfigManager) IsAPIMode() bool {
	return cm.config.OperationMode == ModeAPI
}

// IsLocalMode returns true if the launcher should use local mode
func (cm *ConfigManager) IsLocalMode() bool {
	return cm.config.OperationMode == ModeLocal
}

// IsAutoMode returns true if the launcher should auto-detect the mode
func (cm *ConfigManager) IsAutoMode() bool {
	return cm.config.OperationMode == ModeAuto
}
