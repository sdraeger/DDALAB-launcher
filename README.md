# DDALAB Launcher

A user-friendly Go launcher for DDALAB (Delay Differential Analysis Laboratory) that simplifies deployment and management for non-technical users.

## Features

- 🔍 **Auto-detection**: Automatically finds DDALAB installations on your system
- 💾 **State Persistence**: Remembers your configuration in `~/.ddalab-launcher`
- 🎯 **Simple Interface**: Modern terminal UI using `bubbletea` for intuitive navigation
- 🚀 **One-click Operations**: Start, stop, restart, backup, and update DDALAB
- 🖥️ **Cross-platform**: Works on Linux, macOS, and Windows
- 📊 **Status Monitoring**: Check service health and view logs
- ⚙️ **Easy Configuration**: Configure DDALAB installation path with validation
- ⚡ **Interrupt Support**: Cancel long-running operations with Ctrl+C

## Quick Start

### Prerequisites

- Docker and Docker Compose installed
- An existing DDALAB installation (from DDALAB-setup)

### Installation

1. **Download and build**:
   ```bash
   git clone <repository>
   cd launcher
   make build
   ```

2. **Run the launcher**:
   ```bash
   ./bin/ddalab-launcher
   ```

3. **First-time setup**:
   - The launcher will welcome you and search for DDALAB installations
   - Select an existing installation or configure a custom path
   - Choose whether to start DDALAB immediately

### Usage

The launcher can be run in two ways:

1. **Double-click the executable**: The launcher will automatically open a terminal window
2. **Run from terminal**: `./bin/ddalab-launcher`

After the initial setup, the launcher provides these options:

- **Start DDALAB** - Start all services
- **Stop DDALAB** - Stop all services with confirmation
- **Restart DDALAB** - Restart all services
- **Check Status** - View service status and health
- **View Logs** - Display recent service logs (cancellable with Ctrl+C)
- **Configure Installation** - Change DDALAB installation path
- **Backup Database** - Create a database backup
- **Update DDALAB** - Pull latest images and restart (cancellable with Ctrl+C)
- **Check for Launcher Updates** - Check for and install launcher updates
- **Uninstall DDALAB** - Remove all services and data (with double confirmation)
- **Exit** - Close the launcher

## Project Structure

```
launcher/
├── cmd/launcher/          # Main application entry point
├── internal/app/          # Application logic
├── pkg/
│   ├── config/           # Configuration management
│   ├── commands/         # DDALAB operations
│   ├── detector/         # Installation detection
│   ├── interrupt/        # Signal handling for graceful cancellation
│   └── ui/              # User interface
├── Makefile             # Build automation
└── README.md           # This file
```

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Development mode (auto-rebuild)
make dev
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Vet code
make vet

# Run all pre-commit checks (format, vet, lint)
make pre-commit

# Check if code is properly formatted (for CI)
make check-fmt
```

**Important**: Always run `make pre-commit` before committing changes to ensure code quality.

## Configuration

The launcher stores its configuration in `~/.ddalab-launcher` as JSON:

```json
{
  "ddalab_path": "/path/to/DDALAB-setup",
  "first_run": false,
  "last_operation": "start",
  "version": "1.0.0",
  "auto_update_check": true,
  "update_check_interval_hours": 24,
  "last_update_check": "2023-12-01T10:00:00Z"
}
```

### Auto-Update Settings

The launcher includes automatic update checking:

- **`auto_update_check`**: Enable/disable automatic update checks (default: `true`)
- **`update_check_interval_hours`**: Hours between update checks (default: `24`)
- **`last_update_check`**: Timestamp of last update check

Updates are checked automatically on startup if enabled and the interval has passed. Manual checks are always available through the menu.

## Installation Detection

The launcher searches for DDALAB installations in these locations:

- `~/DDALAB-setup`
- `~/Desktop/DDALAB-setup`
- `~/Downloads/DDALAB-setup`
- `/opt/DDALAB-setup`
- `/usr/local/DDALAB-setup`
- `../DDALAB-setup` (relative to current directory)

An installation is considered valid if it contains:
- `docker-compose.yml`
- `README.md`
- Platform-specific script (`ddalab.sh`, `ddalab.ps1`, or `ddalab.bat`)

## Cross-Platform Support

### Linux/macOS
- Uses `ddalab.sh` script
- Commands executed with `bash`

### Windows
- Prefers `ddalab.ps1` (PowerShell) over `ddalab.bat`
- PowerShell scripts run with bypass execution policy

### macOS Security Notes
- Binaries use `.command` extension for better system recognition
- **Easy setup**: Use the included `install-macos.sh` script:
  ```bash
  ./install-macos.sh
  ```
- **Manual setup**: Remove quarantine flag:
  ```bash
  sudo xattr -rd com.apple.quarantine ddalab-launcher-*.command
  ```
- **Alternative**: Right-click → "Open" → "Open" (bypasses Gatekeeper but shows warning each time)
- This is standard for unsigned open-source applications
- After setup, launcher works normally (double-click or terminal)

## Versioning and Releases

The DDALAB Launcher uses **automatic semantic versioning** with GitHub Actions:

- **Patch increment** (default): `v0.1.0` → `v0.1.1` on every push
- **Minor increment**: Use `[feature]`, `[minor]`, or `feat:` in commit messages
- **Major increment**: Use `[major]`, `[breaking]`, or `breaking:` in commit messages

For detailed versioning control, see [VERSIONING.md](VERSIONING.md).

### Example Version Control
```bash
git commit -m "Fix status check timeout"        # → v0.1.1 (patch)
git commit -m "[feature] Add log filtering"     # → v0.2.0 (minor)  
git commit -m "[breaking] Redesign CLI args"    # → v1.0.0 (major)
```

## Error Handling

The launcher includes comprehensive error handling:

- Installation validation before operations
- Docker availability checks
- Graceful fallbacks for missing components
- User-friendly error messages
- Safe operation confirmations for destructive actions
- Interrupt handling for long-running operations (Ctrl+C support)
- Automatic return to main menu after cancellation

## Dependencies

- **bubbletea**: Modern terminal user interface framework
- **Go standard library**: File operations, exec, etc.

## Building Binaries

Create distribution binaries:

```bash
make build-all
```

This creates binaries for:
- Linux (amd64)
- macOS (amd64, arm64)
- Windows (amd64)

### Platform-Specific Builds

#### macOS App Bundle
```bash
make build-macos-app
```
Creates a double-clickable `.app` bundle for macOS.

#### Release Builds
```bash
./scripts/build-release.sh
```
Creates release archives for all platforms with proper packaging.

## Double-Click Support

The launcher includes special handling for double-click execution:

- **macOS**: Opens Terminal.app and runs the launcher
- **Linux**: Tries common terminal emulators (gnome-terminal, konsole, xterm, etc.)
- **Windows**: Opens Windows Terminal or cmd.exe

If no terminal can be opened, a GUI error dialog is displayed with instructions.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project follows the same license as the main DDALAB project.