package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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

	// Extract binary from archive if needed
	binaryReader, err := u.extractBinaryFromArchive(resp.Body, downloadURL)
	if err != nil {
		return fmt.Errorf("failed to extract binary from archive: %w", err)
	}

	// Use platform-specific update strategy
	if runtime.GOOS == "windows" {
		return u.performWindowsUpdate(currentExe, binaryReader)
	} else {
		return u.performUnixUpdate(currentExe, binaryReader)
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
			// Check for appropriate archive format based on OS
			if currentOS == "windows" && strings.HasSuffix(asset.Name, ".zip") {
				return asset.BrowserDownloadURL, asset.Size
			} else if currentOS != "windows" && strings.HasSuffix(asset.Name, ".tar.gz") {
				return asset.BrowserDownloadURL, asset.Size
			}
		}
	}

	// Fallback to any platform match
	for _, platformString := range platformStrings {
		for _, asset := range assets {
			if strings.Contains(asset.Name, platformString) {
				// Check for appropriate archive format based on OS
				if currentOS == "windows" && strings.HasSuffix(asset.Name, ".zip") {
					return asset.BrowserDownloadURL, asset.Size
				} else if currentOS != "windows" && strings.HasSuffix(asset.Name, ".tar.gz") {
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

// ExtractBinaryFromArchive extracts the binary from a compressed archive (exported for testing)
func (u *Updater) ExtractBinaryFromArchive(archiveReader io.Reader, archiveURL string) (io.Reader, error) {
	return u.extractBinaryFromArchive(archiveReader, archiveURL)
}

// extractBinaryFromArchive extracts the binary from a compressed archive
func (u *Updater) extractBinaryFromArchive(archiveReader io.Reader, archiveURL string) (io.Reader, error) {
	if strings.HasSuffix(archiveURL, ".tar.gz") {
		return u.extractFromTarGz(archiveReader)
	} else if strings.HasSuffix(archiveURL, ".zip") {
		return u.extractFromZip(archiveReader)
	}

	// If it's not an archive, return as-is (raw binary)
	return archiveReader, nil
}

// extractFromTarGz extracts the correct architecture binary from a tar.gz archive
func (u *Updater) extractFromTarGz(reader io.Reader) (io.Reader, error) {
	// Create gzip reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Get current platform info
	currentOS := runtime.GOOS
	currentArch := runtime.GOARCH

	// Build the exact platform string we expect
	expectedPlatformString := fmt.Sprintf("%s-%s", currentOS, currentArch)

	// Alternative patterns we might encounter
	expectedPatterns := []string{
		fmt.Sprintf("ddalab-launcher-%s-%s", currentOS, currentArch),
		fmt.Sprintf("launcher-%s-%s", currentOS, currentArch),
		expectedPlatformString,
	}

	// On Windows, also look for .exe versions
	if currentOS == "windows" {
		windowsPatterns := make([]string, len(expectedPatterns))
		for i, pattern := range expectedPatterns {
			windowsPatterns[i] = pattern + ".exe"
		}
		expectedPatterns = append(expectedPatterns, windowsPatterns...)
	}

	var binaryData []byte
	var foundBinaryName string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		fileName := filepath.Base(header.Name)

		// Check if this binary matches our current platform
		isCorrectBinary := false

		// First, check for exact pattern matches
		for _, pattern := range expectedPatterns {
			if fileName == pattern || strings.Contains(fileName, pattern) {
				isCorrectBinary = true
				foundBinaryName = fileName
				break
			}
		}

		// If no exact match, check if it's a generic launcher binary and contains our platform string
		if !isCorrectBinary {
			if (fileName == "ddalab-launcher" || fileName == "launcher" ||
				(currentOS == "windows" && (fileName == "ddalab-launcher.exe" || fileName == "launcher.exe"))) &&
				strings.Contains(header.Name, expectedPlatformString) {
				isCorrectBinary = true
				foundBinaryName = fileName
			}
		}

		// If this is the correct binary for our platform, extract it
		if isCorrectBinary {
			binaryData = make([]byte, header.Size)
			_, err = io.ReadFull(tarReader, binaryData)
			if err != nil {
				return nil, fmt.Errorf("failed to read binary from archive: %w", err)
			}
			break
		}
	}

	if len(binaryData) == 0 {
		return nil, fmt.Errorf("no binary found for platform %s in archive. Expected patterns: %v",
			expectedPlatformString, expectedPatterns)
	}

	// Validate that we got a reasonable binary size
	if len(binaryData) < 1024 {
		return nil, fmt.Errorf("extracted binary '%s' is too small (%d bytes), likely not a valid executable",
			foundBinaryName, len(binaryData))
	}

	fmt.Printf("Successfully extracted binary '%s' (%d bytes) for platform %s\n",
		foundBinaryName, len(binaryData), expectedPlatformString)

	return bytes.NewReader(binaryData), nil
}

// extractFromZip extracts the correct architecture binary from a ZIP archive
func (u *Updater) extractFromZip(reader io.Reader) (io.Reader, error) {
	// Read all data into memory (required for zip.NewReader)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read ZIP data: %w", err)
	}

	// Create zip reader
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create ZIP reader: %w", err)
	}

	// Get current platform info
	currentOS := runtime.GOOS
	currentArch := runtime.GOARCH

	// Build the exact platform string we expect
	expectedPlatformString := fmt.Sprintf("%s-%s", currentOS, currentArch)

	// Alternative patterns we might encounter
	expectedPatterns := []string{
		fmt.Sprintf("ddalab-launcher-%s-%s.exe", currentOS, currentArch),
		fmt.Sprintf("launcher-%s-%s.exe", currentOS, currentArch),
		fmt.Sprintf("%s.exe", expectedPlatformString),
		"ddalab-launcher.exe",
		"launcher.exe",
	}

	var binaryData []byte
	var foundBinaryName string

	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		fileName := filepath.Base(file.Name)

		// Check if this binary matches our current platform
		isCorrectBinary := false

		// First, check for exact pattern matches
		for _, pattern := range expectedPatterns {
			if fileName == pattern || strings.Contains(fileName, pattern) {
				isCorrectBinary = true
				foundBinaryName = fileName
				break
			}
		}

		// If no exact match, check if it's a generic launcher binary and contains our platform string
		if !isCorrectBinary {
			if (fileName == "ddalab-launcher.exe" || fileName == "launcher.exe") &&
				strings.Contains(file.Name, expectedPlatformString) {
				isCorrectBinary = true
				foundBinaryName = fileName
			}
		}

		// If this is the correct binary for our platform, extract it
		if isCorrectBinary {
			fileReader, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file in ZIP: %w", err)
			}
			defer fileReader.Close()

			binaryData, err = io.ReadAll(fileReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read binary from ZIP: %w", err)
			}
			break
		}
	}

	if len(binaryData) == 0 {
		return nil, fmt.Errorf("no binary found for platform %s in ZIP archive. Expected patterns: %v",
			expectedPlatformString, expectedPatterns)
	}

	// Validate that we got a reasonable binary size
	if len(binaryData) < 1024 {
		return nil, fmt.Errorf("extracted binary '%s' is too small (%d bytes), likely not a valid executable",
			foundBinaryName, len(binaryData))
	}

	fmt.Printf("Successfully extracted binary '%s' (%d bytes) for platform %s\n",
		foundBinaryName, len(binaryData), expectedPlatformString)

	return bytes.NewReader(binaryData), nil
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
