package status

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ddalab/launcher/pkg/commands"
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

// Monitor continuously monitors DDALAB status
type Monitor struct {
	commander     *commands.Commander
	currentStatus Status
	lastCheck     time.Time
	mutex         sync.RWMutex
	refreshRate   time.Duration
	stopChan      chan bool
	running       bool
}

// NewMonitor creates a new status monitor
func NewMonitor(commander *commands.Commander) *Monitor {
	return &Monitor{
		commander:     commander,
		currentStatus: StatusUnknown,
		refreshRate:   5 * time.Second, // Check every 5 seconds
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

// checkStatus performs the actual status check
func (m *Monitor) checkStatus() Status {
	// Use a timeout context for status checks
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Try to get the running status
	running, err := m.checkIsRunning(ctx)
	if err != nil {
		// Check if it's a configuration error (no DDALAB path set)
		if strings.Contains(err.Error(), "not configured") {
			return StatusUnknown
		}
		return StatusError
	}

	if running {
		// Double-check by trying to get service health
		if m.verifyServicesHealthy(ctx) {
			return StatusUp
		} else {
			return StatusStarting // Services are starting but not fully healthy
		}
	}

	return StatusDown
}

// checkIsRunning checks if DDALAB is running with timeout
func (m *Monitor) checkIsRunning(ctx context.Context) (bool, error) {
	// Create a channel to receive the result
	resultChan := make(chan bool, 1)
	errorChan := make(chan error, 1)

	go func() {
		running, err := m.commander.IsRunning()
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- running
		}
	}()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case err := <-errorChan:
		return false, err
	case running := <-resultChan:
		return running, nil
	}
}

// verifyServicesHealthy checks if services are actually healthy
func (m *Monitor) verifyServicesHealthy(ctx context.Context) bool {
	// Create a channel to receive the result
	resultChan := make(chan bool, 1)

	go func() {
		services, err := m.commander.GetServiceHealth()
		if err != nil {
			resultChan <- false
			return
		}

		// Check if core services are healthy
		healthy := true
		coreServices := []string{"ddalab", "postgres", "redis"} // Core services that must be up

		for _, service := range coreServices {
			if status, exists := services[service]; exists {
				// Consider service healthy if it's "Up", "running", or "healthy"
				if !isHealthyStatus(status) {
					healthy = false
					break
				}
			}
		}

		resultChan <- healthy
	}()

	select {
	case <-ctx.Done():
		return false
	case healthy := <-resultChan:
		return healthy
	}
}

// isHealthyStatus determines if a service status indicates health
func isHealthyStatus(status string) bool {
	healthyStatuses := []string{"Up", "running", "healthy", "Up (healthy)"}
	statusLower := strings.ToLower(status)

	for _, healthy := range healthyStatuses {
		if strings.Contains(statusLower, strings.ToLower(healthy)) {
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
