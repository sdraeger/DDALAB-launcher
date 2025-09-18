package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents the API client for Docker extension communication
type Client struct {
	baseURL        string
	httpClient     *http.Client
	apiVersion     string          // Preferred API version
	serverFeatures map[string]bool // Server features from version endpoint
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:        baseURL,
		apiVersion:     "v1", // Default to v1
		serverFeatures: make(map[string]bool),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StandardResponse wraps all API responses from the backend
type StandardResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data,omitempty"`
	Error    *ErrorInfo  `json:"error,omitempty"`
	Metadata *Metadata   `json:"metadata"`
}

// ErrorInfo provides detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Metadata provides response metadata
type Metadata struct {
	Timestamp     string `json:"timestamp"`
	APIVersion    string `json:"api_version"`
	ServerVersion string `json:"server_version"`
}

// Status represents the DDALAB status response
type Status struct {
	Running      bool             `json:"running"`
	State        string           `json:"state"`
	Services     []Service        `json:"services"`
	Installation InstallationInfo `json:"installation"`
}

// Service represents a single service status
type Service struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Health string `json:"health"`
	Uptime string `json:"uptime,omitempty"`
}

// InstallationInfo represents installation details
type InstallationInfo struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	LastUpdated string `json:"last_updated"`
	Valid       bool   `json:"valid"`
}

// EnvConfig represents environment configuration
type EnvConfig struct {
	URL    string `json:"url"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	Scheme string `json:"scheme"`
	Domain string `json:"domain"`
}

// PathValidationResult represents path validation response
type PathValidationResult struct {
	Valid           bool   `json:"valid"`
	Path            string `json:"path"`
	Message         string `json:"message"`
	HasCompose      bool   `json:"has_compose"`
	HasDDALABScript bool   `json:"has_ddalab_script"`
}

// VersionInfo represents API version information
type VersionInfo struct {
	Version            string          `json:"version"`
	APIVersion         string          `json:"api_version"`
	SupportedVersions  []string        `json:"supported_versions"`
	DeprecatedVersions []string        `json:"deprecated_versions"`
	Server             string          `json:"server"`
	Features           map[string]bool `json:"features"`
}

// HealthCheck function to verify API availability
func (c *Client) HealthCheck(ctx context.Context) error {
	// First try to get version info to validate compatibility
	if err := c.checkVersion(ctx); err != nil {
		// If version check fails, fall back to basic health check
		return c.basicHealthCheck(ctx)
	}
	return nil
}

// checkVersion retrieves and validates API version compatibility
func (c *Client) checkVersion(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/version", nil)
	if err != nil {
		return fmt.Errorf("failed to create version request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("version check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("version check failed with status: %d", resp.StatusCode)
	}

	var versionInfo VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return fmt.Errorf("failed to decode version response: %w", err)
	}

	// Check if our preferred version is supported
	supported := false
	for _, supportedVersion := range versionInfo.SupportedVersions {
		if supportedVersion == c.apiVersion {
			supported = true
			break
		}
	}

	if !supported {
		// Try to use the latest supported version
		if len(versionInfo.SupportedVersions) > 0 {
			c.apiVersion = versionInfo.SupportedVersions[0]
		} else {
			return fmt.Errorf("no supported API versions found")
		}
	}

	// Store server features for capability checks
	c.serverFeatures = versionInfo.Features

	return nil
}

// basicHealthCheck performs a simple health check without version validation
func (c *Client) basicHealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/test", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetStatus retrieves the current DDALAB status using the new v1 API
func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	endpoint := fmt.Sprintf("/api/%s/status", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status request failed with status: %d", resp.StatusCode)
	}

	var response StandardResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	if !response.Success {
		if response.Error != nil {
			return nil, fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	// Convert the data to Status struct
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status data: %w", err)
	}

	var status Status
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status data: %w", err)
	}

	return &status, nil
}

// StartStack starts all DDALAB services using the new lifecycle API
func (c *Client) StartStack(ctx context.Context) error {
	return c.lifecycleAction(ctx, "start")
}

// StopStack stops all DDALAB services using the new lifecycle API
func (c *Client) StopStack(ctx context.Context) error {
	return c.lifecycleAction(ctx, "stop")
}

// RestartStack restarts all DDALAB services using the new lifecycle API
func (c *Client) RestartStack(ctx context.Context) error {
	return c.lifecycleAction(ctx, "restart")
}

// UpdateStack updates DDALAB using the new lifecycle API
func (c *Client) UpdateStack(ctx context.Context) error {
	return c.lifecycleAction(ctx, "update")
}

// lifecycleAction performs a lifecycle action using the new v1 API
func (c *Client) lifecycleAction(ctx context.Context, action string) error {
	endpoint := fmt.Sprintf("/api/%s/lifecycle/%s", c.apiVersion, action)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create %s request: %w", action, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s failed with status %d: %s", action, resp.StatusCode, string(body))
	}

	// Parse the standardized response
	var response StandardResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode %s response: %w", action, err)
	}

	if !response.Success {
		if response.Error != nil {
			return fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
		}
		return fmt.Errorf("%s operation failed", action)
	}

	return nil
}

// GetLogs retrieves service logs using the new v1 API
func (c *Client) GetLogs(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("/api/%s/logs", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create logs request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("logs request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("logs request failed with status: %d", resp.StatusCode)
	}

	var response StandardResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode logs response: %w", err)
	}

	if !response.Success {
		if response.Error != nil {
			return "", fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
		}
		return "", fmt.Errorf("logs request failed")
	}

	// Extract logs from the response data
	if data, ok := response.Data.(map[string]interface{}); ok {
		if logs, exists := data["logs"]; exists {
			if logStr, ok := logs.(string); ok {
				return logStr, nil
			}
		}
	}

	return "", fmt.Errorf("unexpected logs response format")
}

// CreateBackup creates a database backup using legacy endpoint
func (c *Client) CreateBackup(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/backup", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create backup request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("backup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("backup failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode backup response: %w", err)
	}

	return result["filename"], nil
}

// UpdateDDALAB updates DDALAB to the latest version (legacy method - use UpdateStack instead)
func (c *Client) UpdateDDALAB(ctx context.Context) error {
	return c.UpdateStack(ctx)
}

// GetEnvConfig retrieves environment configuration
func (c *Client) GetEnvConfig(ctx context.Context) (*EnvConfig, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/env", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create env config request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("env config request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("env config request failed with status: %d", resp.StatusCode)
	}

	var config EnvConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode env config response: %w", err)
	}

	return &config, nil
}

// ValidatePath validates a DDALAB installation path using v1 API
func (c *Client) ValidatePath(ctx context.Context, path string) (*PathValidationResult, error) {
	payload := map[string]string{"path": path}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal path validation request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/%s/paths/validate", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create path validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("path validation request failed: %w", err)
	}
	defer resp.Body.Close()

	var result PathValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode path validation response: %w", err)
	}

	return &result, nil
}

// SelectPath selects a DDALAB installation path using v1 API
func (c *Client) SelectPath(ctx context.Context, path string) error {
	payload := map[string]string{"path": path}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal path selection request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/%s/paths/select", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create path selection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("path selection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("path selection failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DiscoverPaths discovers DDALAB installation paths
func (c *Client) DiscoverPaths(ctx context.Context) ([]string, error) {
	endpoint := fmt.Sprintf("/api/%s/paths/discover", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create path discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("path discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("path discovery request failed with status: %d", resp.StatusCode)
	}

	var result map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode path discovery response: %w", err)
	}

	if paths, exists := result["discovered_paths"]; exists {
		return paths, nil
	}

	return []string{}, nil
}

// Environment Configuration API methods

// EnvVariable represents a single environment variable
type EnvVariable struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Comment    string `json:"comment,omitempty"`
	Section    string `json:"section,omitempty"`
	IsRequired bool   `json:"is_required"`
	IsSecret   bool   `json:"is_secret"`
}

// EnvConfigResponse represents the complete environment configuration response
type EnvConfigResponse struct {
	Config       *EnvConfigData           `json:"config"`
	FilePath     string                   `json:"file_path"`
	FileExists   bool                     `json:"file_exists"`
	LastModified string                   `json:"last_modified,omitempty"`
	Sections     map[string][]EnvVariable `json:"sections"`
	Summary      *ConfigSummary           `json:"summary"`
}

// EnvConfigData represents the environment configuration data
type EnvConfigData struct {
	Variables []EnvVariable `json:"variables"`
	FilePath  string        `json:"file_path"`
	Sections  []string      `json:"sections"`
}

// ConfigSummary provides overview statistics
type ConfigSummary struct {
	TotalVariables    int `json:"total_variables"`
	RequiredVariables int `json:"required_variables"`
	SecretVariables   int `json:"secret_variables"`
	EmptyVariables    int `json:"empty_variables"`
	SectionCount      int `json:"section_count"`
}

// GetEnvConfigNew retrieves environment configuration using the new v1 API
func (c *Client) GetEnvConfigNew(ctx context.Context) (*EnvConfigResponse, error) {
	endpoint := fmt.Sprintf("/api/%s/config/env", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create env config request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("env config request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("env config request failed with status: %d", resp.StatusCode)
	}

	var response StandardResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode env config response: %w", err)
	}

	if !response.Success {
		if response.Error != nil {
			return nil, fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
		}
		return nil, fmt.Errorf("env config request failed")
	}

	// Convert the data to EnvConfigResponse struct
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal env config data: %w", err)
	}

	var envConfig EnvConfigResponse
	if err := json.Unmarshal(dataBytes, &envConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal env config data: %w", err)
	}

	return &envConfig, nil
}

// UpdateEnvConfig updates environment configuration using the new v1 API
func (c *Client) UpdateEnvConfig(ctx context.Context, variables []EnvVariable) error {
	payload := map[string]interface{}{
		"variables":     variables,
		"create_backup": true,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal env config update request: %w", err)
	}

	endpoint := fmt.Sprintf("/api/%s/config/env", c.apiVersion)
	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create env config update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("env config update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("env config update failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
