package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/inconshreveable/go-update"
)

const (
	GitHubRepoOwner = "sdraeger"
	GitHubRepoName  = "DDALAB-launcher"
	UpdateCheckURL  = "https://api.github.com/repos/sdraeger/DDALAB-launcher/releases/latest"
)

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
	PublishedAt time.Time `json:"published_at"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	DownloadURL    string
	Size           int64
	PublishedAt    time.Time
	HasUpdate      bool
}

// Updater handles launcher self-updates
type Updater struct {
	currentVersion string
	githubToken    string // Optional for rate limiting
}

// NewUpdater creates a new updater instance
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		githubToken:    os.Getenv("GITHUB_TOKEN"), // Optional
	}
}

// CheckForUpdates checks if a new version is available
func (u *Updater) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", UpdateCheckURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if available (helps with rate limiting)
	if u.githubToken != "" {
		req.Header.Set("Authorization", "token "+u.githubToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	// Parse versions
	currentVer, err := u.parseVersion(u.currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current version: %w", err)
	}

	latestVer, err := u.parseVersion(release.TagName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse latest version: %w", err)
	}

	// Find the appropriate binary for current platform
	downloadURL, size := u.findPlatformBinary(release.Assets)

	updateInfo := &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		DownloadURL:    downloadURL,
		Size:           size,
		PublishedAt:    release.PublishedAt,
		HasUpdate:      latestVer.GT(currentVer),
	}

	return updateInfo, nil
}

// PerformUpdate downloads and applies the update safely
func (u *Updater) PerformUpdate(ctx context.Context, downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download URL available for this platform")
	}

	// Get the path to the current executable
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Resolve any symlinks to get the real path
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Download the new binary
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Use platform-specific update strategy
	if runtime.GOOS == "windows" {
		return u.performWindowsUpdate(currentExe, resp.Body)
	} else {
		return u.performUnixUpdate(currentExe, resp.Body)
	}
}

// ParseVersion parses a version string, handling 'v' prefix (exported for testing)
func (u *Updater) ParseVersion(version string) (semver.Version, error) {
	return u.parseVersion(version)
}

// parseVersion parses a version string, handling 'v' prefix
func (u *Updater) parseVersion(version string) (semver.Version, error) {
	// Remove 'v' prefix if present
	cleanVersion := strings.TrimPrefix(version, "v")

	// Handle 'dev' version
	if cleanVersion == "dev" {
		return semver.Version{Major: 0, Minor: 0, Patch: 0}, nil
	}

	return semver.Parse(cleanVersion)
}

// findPlatformBinary finds the appropriate binary for the current platform
func (u *Updater) findPlatformBinary(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}) (string, int64) {
	platformMap := map[string][]string{
		"darwin":  {"darwin-amd64", "darwin-arm64"},
		"linux":   {"linux-amd64", "linux-arm64"},
		"windows": {"windows-amd64", "windows-arm64"},
	}

	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}

	currentOS := runtime.GOOS
	currentArch := runtime.GOARCH

	// Look for platform-specific binaries
	platformStrings, exists := platformMap[currentOS]
	if !exists {
		return "", 0
	}

	archString, exists := archMap[currentArch]
	if !exists {
		archString = "amd64" // Default fallback
	}

	// Try exact match first (OS-ARCH)
	for _, asset := range assets {
		expectedName := fmt.Sprintf("%s-%s", currentOS, archString)
		if strings.Contains(asset.Name, expectedName) {
			// Skip checksums and other files
			if strings.HasSuffix(asset.Name, ".tar.gz") || strings.HasSuffix(asset.Name, ".zip") {
				return asset.BrowserDownloadURL, asset.Size
			}
		}
	}

	// Fallback to any platform match
	for _, platformString := range platformStrings {
		for _, asset := range assets {
			if strings.Contains(asset.Name, platformString) {
				if strings.HasSuffix(asset.Name, ".tar.gz") || strings.HasSuffix(asset.Name, ".zip") {
					return asset.BrowserDownloadURL, asset.Size
				}
			}
		}
	}

	return "", 0
}

// FormatSize formats byte size in human readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ShouldCheckForUpdates determines if we should check for updates based on last check time
func ShouldCheckForUpdates(lastCheckTime time.Time, interval time.Duration) bool {
	return time.Since(lastCheckTime) >= interval
}

// GetPlatformString returns a human-readable platform string
func GetPlatformString() string {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	osMap := map[string]string{
		"darwin":  "macOS",
		"linux":   "Linux",
		"windows": "Windows",
	}

	archMap := map[string]string{
		"amd64": "x64",
		"arm64": "ARM64",
		"386":   "x86",
	}

	if displayOS, exists := osMap[osName]; exists {
		osName = displayOS
	}

	if displayArch, exists := archMap[archName]; exists {
		archName = displayArch
	}

	return fmt.Sprintf("%s %s", osName, archName)
}

// performUnixUpdate handles updates on Unix-like systems (macOS, Linux)
func (u *Updater) performUnixUpdate(currentExe string, updateBody io.Reader) error {
	// Create a temporary file for the new binary
	tempDir := filepath.Dir(currentExe)
	tempFile, err := os.CreateTemp(tempDir, "launcher-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Ensure cleanup
	defer func() {
		_ = os.Remove(tempPath)
	}()

	// Write the new binary to temp file using go-update (for validation and verification)
	err = update.Apply(updateBody, update.Options{
		TargetPath: tempPath,
	})
	if err != nil {
		return fmt.Errorf("failed to apply update to temporary file: %w", err)
	}

	// Make the temp file executable
	err = os.Chmod(tempPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to make temporary file executable: %w", err)
	}

	// Create backup of current binary
	backupPath := currentExe + ".backup"
	err = os.Rename(currentExe, backupPath)
	if err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	err = os.Rename(tempPath, currentExe)
	if err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to move new binary into place: %w", err)
	}

	// Remove backup on success
	_ = os.Remove(backupPath)

	return nil
}

// performWindowsUpdate handles updates on Windows
func (u *Updater) performWindowsUpdate(currentExe string, updateBody io.Reader) error {
	// On Windows, we can't replace a running executable directly
	// We use a different strategy: download to .new, create a batch script to replace it

	newPath := currentExe + ".new"
	batchPath := currentExe + ".update.bat"

	// Ensure cleanup
	defer func() {
		_ = os.Remove(newPath)
		_ = os.Remove(batchPath)
	}()

	// Write the new binary using go-update
	err := update.Apply(updateBody, update.Options{
		TargetPath: newPath,
	})
	if err != nil {
		return fmt.Errorf("failed to apply update to .new file: %w", err)
	}

	// Create a batch script to perform the replacement after this process exits
	batchContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move "%s" "%s.old"
move "%s" "%s"
del "%s.old"
del "%%~f0"
`, currentExe, currentExe, newPath, currentExe, currentExe)

	err = os.WriteFile(batchPath, []byte(batchContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create update batch script: %w", err)
	}

	// Start the batch script in the background
	cmd := exec.Command("cmd", "/c", "start", "/b", batchPath)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start update batch script: %w", err)
	}

	return nil
}
