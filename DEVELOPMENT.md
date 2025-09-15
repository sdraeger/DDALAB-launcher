# Development Guide

## Getting Started

### Prerequisites
- Go 1.21 or later
- Git
- golangci-lint (optional but recommended)
- CGO-compatible C compiler (for GUI functionality)
  - macOS: Xcode Command Line Tools
  - Linux: gcc, pkg-config, OpenGL/mesa headers
  - Windows: TDM-GCC or similar

### Setup Development Environment

1. **Clone and navigate to the project:**
```bash
git clone <repository-url>
cd launcher
```

2. **Install dependencies:**
```bash
make deps
```

3. **Setup Git hooks (recommended):**
```bash
make setup-hooks
```

This will install pre-commit hooks that automatically:
- Format your code with `gofmt`
- Run `go vet` for code quality checks
- Run `golangci-lint` (if available)
- Run all tests

## Development Workflow

### Building
```bash
# Build for current platform (with GUI support)
make build

# Build without GUI (for CI/headless environments)  
make build-nogui

# Build for all platforms
make build-all

# Build macOS app bundle
make build-macos-app
```

### Testing
```bash
# Run all tests (automatically excludes GUI to avoid CGO dependencies)
make test

# Run tests with coverage
make test-coverage
```

**Note:** Tests automatically run in no-GUI mode to avoid CGO dependencies in CI environments. The GUI functionality is tested through manual verification.

### Code Quality

#### Manual Checks
```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Run all pre-commit checks
make pre-commit

# Quick checks (without linter)
make pre-commit-quick
```

#### Automatic Checks
If you've run `make setup-hooks`, these checks will run automatically before each commit:

1. **Code Formatting**: Automatically formats Go code
2. **Code Validation**: Runs `go vet` to catch potential issues
3. **Linting**: Runs `golangci-lint` if available
4. **Tests**: Ensures all tests pass

### Bypassing Pre-commit Hooks
For emergency commits, you can bypass the hooks with:
```bash
git commit --no-verify
```

**Note:** Only use this in emergencies. The CI will still catch formatting and linting issues.

## Installing golangci-lint

To get the full benefit of the pre-commit hooks, install golangci-lint:

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

Or on macOS with Homebrew:
```bash
brew install golangci-lint
```

## Project Structure

```
launcher/
├── cmd/launcher/           # Main application entry point
├── internal/app/           # Application logic
├── pkg/                    # Reusable packages
│   ├── commands/          # DDALAB command execution
│   ├── config/            # Configuration management
│   ├── detector/          # Installation detection
│   ├── gui/               # Experimental GUI (Fyne-based)
│   ├── interrupt/         # Signal handling
│   ├── status/            # Status monitoring
│   ├── ui/                # User interface (TUI)
│   └── updater/           # Self-update functionality
├── scripts/               # Build and utility scripts
├── build/                 # Build configuration files
└── Makefile              # Build automation
```

## Common Tasks

### Adding a New Feature
1. Create your feature branch: `git checkout -b feature/my-feature`
2. Write your code following existing patterns
3. Add tests for your functionality
4. Run `make pre-commit` to ensure code quality
5. Commit your changes (pre-commit hooks will run automatically)
6. Create a pull request

### Debugging
```bash
# Run in development mode
make dev

# Or build and run with debug info
make build
./bin/ddalab-launcher -version
```

### GUI Development
The launcher includes an experimental GUI built with Fyne:

```bash
# GUI requires CGO to be enabled (automatically handled by Makefile)
make build

# Test GUI functionality
./bin/ddalab-launcher
# Then select "Open GUI (Experimental)" from the menu

# For headless environments or CI, use the no-GUI build:
make build-nogui
./bin/ddalab-launcher-nogui
# GUI option will show an error message explaining GUI is not available
```

**GUI Features:**
- Service control (Start/Stop/Restart)
- Real-time status monitoring
- Log viewing in separate windows
- Configuration management
- Update checking and installation
- Cross-platform compatibility

**Build Tags:**
- Default build: Includes GUI support (requires CGO)
- `nogui` tag: Excludes GUI, no CGO dependencies (used for CI/testing)

### Release Process
1. Update version in build scripts
2. Run `make build-all` to build for all platforms
3. Test on target platforms
4. Create release tags and packages

## Troubleshooting

### Pre-commit Hook Issues
- If the hook fails, fix the reported issues and commit again
- Check that you have all required tools installed
- Run `make pre-commit` manually to debug

### Build Issues
- Ensure Go version is 1.21+
- Run `make deps` to update dependencies
- Check that all imports are available

### Linting Issues
- Install golangci-lint for local development
- Use `make fmt` to fix formatting issues
- Check the CI logs for detailed error messages