package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Version is injected at build time
var Version = "dev"

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// LauncherConfig holds the persistent state of the launcher
type LauncherConfig struct {
	DDALABPath    string `json:"ddalab_path"`
	FirstRun      bool   `json:"first_run"`
	LastOperation string `json:"last_operation"`
	Version       string `json:"version"`
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
			FirstRun: true,
			Version:  GetVersion(),
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