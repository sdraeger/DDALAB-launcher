package mode

import (
	"context"
	"fmt"
	"time"

	"github.com/ddalab/launcher/pkg/api"
	"github.com/ddalab/launcher/pkg/bootstrap"
	"github.com/ddalab/launcher/pkg/config"
)

// Manager handles operation mode detection and switching
type Manager struct {
	configManager *config.ConfigManager
	apiClient     *api.Client
	currentMode   config.OperationMode
	bootstrapper  *bootstrap.Bootstrap
}

// NewManager creates a new mode manager
func NewManager(configManager *config.ConfigManager) *Manager {
	apiClient := api.NewClient(configManager.GetAPIEndpoint())
	bootstrapper := bootstrap.NewBootstrap()

	return &Manager{
		configManager: configManager,
		apiClient:     apiClient,
		currentMode:   config.ModeLocal, // Start with local mode as fallback
		bootstrapper:  bootstrapper,
	}
}

// Initialize determines and sets the appropriate operation mode
func (m *Manager) Initialize() error {
	// First, check Docker extension availability
	if err := m.bootstrapper.CheckDockerExtension(); err != nil {
		// Log the bootstrap check result but don't fail initialization
		// The launcher can still work in local mode
		_ = err
	}

	configuredMode := m.configManager.GetOperationMode()

	switch configuredMode {
	case config.ModeLocal:
		// Local mode is deprecated, treat as auto mode
		m.currentMode = m.detectBestMode()
		return nil
	case config.ModeAPI:
		if err := m.verifyAPIMode(); err != nil {
			// If API mode fails but bootstrap is available, try bootstrap
			if m.bootstrapper.CanBootstrap() {
				if bootstrapErr := m.tryBootstrapAPI(); bootstrapErr == nil {
					m.currentMode = config.ModeAPI
					return nil
				}
			}
			return fmt.Errorf("API mode configured but not available: %w", err)
		}
		m.currentMode = config.ModeAPI
		return nil
	case config.ModeAuto:
		m.currentMode = m.detectBestMode()
		return nil
	default:
		// Fallback to auto mode for unknown configurations
		m.currentMode = m.detectBestMode()
		return nil
	}
}

// detectBestMode automatically detects the best operation mode
func (m *Manager) detectBestMode() config.OperationMode {
	// Try API mode first (preferred if available)
	if err := m.verifyAPIMode(); err == nil {
		return config.ModeAPI
	}

	// If API is not available but we can bootstrap, try that
	if m.bootstrapper.CanBootstrap() {
		if err := m.tryBootstrapAPI(); err == nil {
			// Re-verify API mode after bootstrap attempt
			if verifyErr := m.verifyAPIMode(); verifyErr == nil {
				return config.ModeAPI
			}
		}
	}

	// Fallback to local mode
	return config.ModeLocal
}

// tryBootstrapAPI attempts to bootstrap the API backend
func (m *Manager) tryBootstrapAPI() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First try to start the extension backend if available
	if m.bootstrapper.IsExtensionAvailable() {
		if err := m.bootstrapper.StartExtensionBackend(ctx); err == nil {
			return nil
		}
	}

	// If that fails or is not available, try minimal services
	ddalabPath := m.configManager.GetDDALABPath()
	if ddalabPath == "" {
		return fmt.Errorf("DDALAB path not configured")
	}

	return m.bootstrapper.StartMinimalServices(ctx, ddalabPath)
}

// verifyAPIMode checks if the API mode is available
func (m *Manager) verifyAPIMode() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.apiClient.HealthCheck(ctx)
}

// GetCurrentMode returns the current operation mode
func (m *Manager) GetCurrentMode() config.OperationMode {
	return m.currentMode
}

// IsAPIMode returns true if currently in API mode
func (m *Manager) IsAPIMode() bool {
	return m.currentMode == config.ModeAPI
}

// IsLocalMode returns true if currently in local mode (deprecated, always returns false)
func (m *Manager) IsLocalMode() bool {
	// Local mode is deprecated in the new architecture
	return false
}

// GetAPIClient returns the API client (only valid in API mode)
func (m *Manager) GetAPIClient() *api.Client {
	if !m.IsAPIMode() {
		return nil
	}
	return m.apiClient
}

// SwitchMode switches to a specific operation mode
func (m *Manager) SwitchMode(newMode config.OperationMode) error {
	switch newMode {
	case config.ModeAPI:
		if err := m.verifyAPIMode(); err != nil {
			return fmt.Errorf("cannot switch to API mode: %w", err)
		}
		m.currentMode = config.ModeAPI
		m.configManager.SetOperationMode(config.ModeAPI)
	case config.ModeLocal:
		m.currentMode = config.ModeLocal
		m.configManager.SetOperationMode(config.ModeLocal)
	case config.ModeAuto:
		m.currentMode = m.detectBestMode()
		m.configManager.SetOperationMode(config.ModeAuto)
	default:
		return fmt.Errorf("unknown operation mode: %s", newMode)
	}

	// Save the configuration
	return m.configManager.Save()
}

// GetModeStatus returns detailed information about the current mode
func (m *Manager) GetModeStatus() ModeStatus {
	status := ModeStatus{
		CurrentMode:        m.currentMode,
		ConfiguredMode:     m.configManager.GetOperationMode(),
		BootstrapMode:      m.bootstrapper.GetBootstrapMode(),
		CanBootstrap:       m.bootstrapper.CanBootstrap(),
		ExtensionAvailable: m.bootstrapper.IsExtensionAvailable(),
	}

	// Check API availability
	if err := m.verifyAPIMode(); err == nil {
		status.APIAvailable = true
		status.APIEndpoint = m.configManager.GetAPIEndpoint()
	} else {
		status.APIAvailable = false
		status.APIError = err.Error()
	}

	return status
}

// ModeStatus provides detailed information about operation modes
type ModeStatus struct {
	CurrentMode        config.OperationMode `json:"current_mode"`
	ConfiguredMode     config.OperationMode `json:"configured_mode"`
	APIAvailable       bool                 `json:"api_available"`
	APIEndpoint        string               `json:"api_endpoint,omitempty"`
	APIError           string               `json:"api_error,omitempty"`
	BootstrapMode      string               `json:"bootstrap_mode"`
	CanBootstrap       bool                 `json:"can_bootstrap"`
	ExtensionAvailable bool                 `json:"extension_available"`
}

// GetModeDescription returns a human-readable description of the mode
func (m *Manager) GetModeDescription() string {
	switch m.currentMode {
	case config.ModeAPI:
		return "Using Docker Extension API for DDALAB management"
	case config.ModeLocal:
		return "Deprecated local mode - switching to API with bootstrap fallback"
	default:
		return "API mode with automatic bootstrap fallback"
	}
}

// RefreshMode re-evaluates the current mode (useful for auto mode)
func (m *Manager) RefreshMode() error {
	if m.configManager.IsAutoMode() {
		newMode := m.detectBestMode()
		if newMode != m.currentMode {
			m.currentMode = newMode
		}
	}
	return nil
}

// GetBootstrapper returns the bootstrap instance for direct access
func (m *Manager) GetBootstrapper() *bootstrap.Bootstrap {
	return m.bootstrapper
}

// PerformBootstrap attempts to bootstrap DDALAB services and switch to API mode
func (m *Manager) PerformBootstrap() error {
	if !m.bootstrapper.CanBootstrap() {
		return fmt.Errorf("bootstrap not available - Docker is not running")
	}

	// Try to bootstrap the API
	if err := m.tryBootstrapAPI(); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Wait for services to be ready
	time.Sleep(5 * time.Second)

	// Verify API is now available
	if err := m.verifyAPIMode(); err != nil {
		return fmt.Errorf("bootstrap appeared to succeed but API is not available: %w", err)
	}

	// Switch to API mode
	m.currentMode = config.ModeAPI
	return nil
}
