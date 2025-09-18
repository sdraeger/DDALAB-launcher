package status

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ddalab/launcher/pkg/api"
)

// Status represents the current DDALAB status
type Status int

const (
	StatusUnknown Status = iota
	StatusUp
	StatusDown
	StatusStarting
	StatusStopping
	StatusError
)

// String returns a human-readable status string
func (s Status) String() string {
	switch s {
	case StatusUp:
		return "Up"
	case StatusDown:
		return "Down"
	case StatusStarting:
		return "Starting"
	case StatusStopping:
		return "Stopping"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// GetColoredDot returns a colored dot for the status
func (s Status) GetColoredDot() string {
	switch s {
	case StatusUp:
		return "🟢" // Green dot
	case StatusDown:
		return "🔴" // Red dot
	case StatusStarting:
		return "🟡" // Yellow dot
	case StatusStopping:
		return "🟡" // Yellow dot
	case StatusError:
		return "🔴" // Red dot
	default:
		return "⚪" // White dot
	}
}

// Monitor continuously monitors DDALAB status via API
type Monitor struct {
	apiClient     *api.Client
	currentStatus Status
	lastCheck     time.Time
	mutex         sync.RWMutex
	refreshRate   time.Duration
	stopChan      chan bool
	running       bool
}

// NewMonitor creates a new status monitor that uses the API client
func NewMonitor(apiClient *api.Client) *Monitor {
	return &Monitor{
		apiClient:     apiClient,
		currentStatus: StatusUnknown,
		refreshRate:   1 * time.Second, // Check every 1 second for real-time updates
		stopChan:      make(chan bool),
	}
}

// Start begins monitoring DDALAB status in the background
func (m *Monitor) Start() {
	m.mutex.Lock()
	if m.running {
		m.mutex.Unlock()
		return
	}
	m.running = true
	m.mutex.Unlock()

	go m.monitorLoop()
}

// Stop stops the background monitoring
func (m *Monitor) Stop() {
	m.mutex.Lock()
	if !m.running {
		m.mutex.Unlock()
		return
	}
	m.running = false
	m.mutex.Unlock()

	select {
	case m.stopChan <- true:
	default:
	}
}

// IsRunning returns true if the monitor is currently running
func (m *Monitor) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}

// GetStatus returns the current status
func (m *Monitor) GetStatus() Status {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.currentStatus
}

// GetLastCheck returns when the status was last checked
func (m *Monitor) GetLastCheck() time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.lastCheck
}

// CheckNow forces an immediate status check
func (m *Monitor) CheckNow() Status {
	status := m.checkStatus()

	m.mutex.Lock()
	m.currentStatus = status
	m.lastCheck = time.Now()
	m.mutex.Unlock()

	return status
}

// FormatStatus returns a formatted status string for display
func (m *Monitor) FormatStatus() string {
	status := m.GetStatus()
	lastCheck := m.GetLastCheck()

	statusText := status.GetColoredDot() + " " + status.String()

	// Add last check time for non-unknown status
	if status != StatusUnknown && !lastCheck.IsZero() {
		// Only show time if it's recent (less than 1 minute old)
		if time.Since(lastCheck) < time.Minute {
			statusText += " (live)"
		} else {
			statusText += fmt.Sprintf(" (%ds ago)", int(time.Since(lastCheck).Seconds()))
		}
	}

	return statusText
}

// monitorLoop runs the background monitoring
func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(m.refreshRate)
	defer ticker.Stop()

	// Do an initial check
	m.CheckNow()

	for {
		select {
		case <-ticker.C:
			m.CheckNow()
		case <-m.stopChan:
			return
		}
	}
}

// checkStatus performs the actual status check using the API
func (m *Monitor) checkStatus() Status {
	// Use a timeout context for status checks
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get status from the API
	status, err := m.apiClient.GetStatus(ctx)
	if err != nil {
		// Check if it's a connection error (backend not available)
		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "connection timeout") {
			return StatusUnknown // Backend not available
		}
		return StatusError
	}

	// Convert API status to local status
	return m.convertAPIStatus(status)
}

// convertAPIStatus converts API status response to local Status enum
func (m *Monitor) convertAPIStatus(apiStatus *api.Status) Status {
	if !apiStatus.Running {
		return StatusDown
	}

	// Check the overall state from the API
	switch strings.ToLower(apiStatus.State) {
	case "up":
		return StatusUp
	case "down":
		return StatusDown
	case "starting":
		return StatusStarting
	case "stopping":
		return StatusStopping
	case "error":
		return StatusError
	default:
		// Fall back to service-level analysis
		return m.analyzeServiceHealth(apiStatus.Services)
	}
}

// analyzeServiceHealth analyzes individual service statuses
func (m *Monitor) analyzeServiceHealth(services []api.Service) Status {
	if len(services) == 0 {
		return StatusDown
	}

	healthyCount := 0
	totalCount := len(services)
	hasErrors := false

	for _, service := range services {
		switch strings.ToLower(service.Health) {
		case "healthy":
			healthyCount++
		case "unhealthy":
			hasErrors = true
		case "starting":
			// Service is starting, don't count as healthy yet
		default:
			// Check legacy status field
			if isHealthyServiceStatus(service.Status) {
				healthyCount++
			} else if isErrorServiceStatus(service.Status) {
				hasErrors = true
			}
		}
	}

	if hasErrors {
		return StatusError
	}

	if healthyCount == totalCount {
		return StatusUp
	}

	if healthyCount > 0 {
		return StatusStarting // Some services healthy, others starting
	}

	return StatusStarting // All services starting
}

// isHealthyServiceStatus determines if a service status indicates health
func isHealthyServiceStatus(status string) bool {
	healthyStatuses := []string{"running", "up", "healthy"}
	statusLower := strings.ToLower(status)

	for _, healthy := range healthyStatuses {
		if strings.Contains(statusLower, healthy) {
			return true
		}
	}
	return false
}

// isErrorServiceStatus determines if a service status indicates an error
func isErrorServiceStatus(status string) bool {
	errorStatuses := []string{"error", "failed", "exited", "dead"}
	statusLower := strings.ToLower(status)

	for _, errorStatus := range errorStatuses {
		if strings.Contains(statusLower, errorStatus) {
			return true
		}
	}
	return false
}

// SetRefreshRate changes how often the status is checked
func (m *Monitor) SetRefreshRate(rate time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if rate < time.Second {
		rate = time.Second // Minimum 1 second
	}

	m.refreshRate = rate
}
