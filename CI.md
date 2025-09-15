# CI/CD Configuration

## Overview

The launcher uses GitHub Actions for continuous integration and testing. The configuration handles both GUI and headless builds to ensure compatibility across different environments.

## Workflows

### Test Workflow (`.github/workflows/test.yml`)

**Main Test Job (`test`):**
- Runs on Ubuntu in headless environment
- Uses `nogui` build tag to avoid CGO/GUI dependencies
- Tests core functionality without GUI requirements
- Performs cross-platform build tests (Linux, macOS, Windows)

**GUI Build Test Job (`test-gui-build`):**
- Installs GUI dependencies on Ubuntu
- Tests that GUI version compiles correctly
- Ensures GUI code doesn't have compilation issues
- Does not run GUI (no display available in CI)

## Build Tags

### `nogui` Tag
- **Purpose**: Excludes GUI code from compilation
- **Benefits**: No CGO dependencies, faster builds, CI-compatible
- **Usage**: Automatically used in tests and CI
- **Functionality**: Provides stub implementation with informative error

### Default Build (no tags)
- **Purpose**: Full GUI support for end users
- **Requirements**: CGO, OpenGL/X11 headers, display system
- **Usage**: Local development and user deployments

## CI Commands

The CI uses these specific commands to avoid GUI dependencies:

```bash
# Tests (no CGO, no GUI dependencies)
CGO_ENABLED=0 go test -tags nogui -v ./...

# Cross-platform builds (headless)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags nogui ./cmd/launcher
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags nogui ./cmd/launcher  
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags nogui ./cmd/launcher

# GUI compilation test (with dependencies)
CGO_ENABLED=1 go build ./cmd/launcher
```

## Local Development

Developers can use the same commands locally:

```bash
# Run tests like CI does
make test

# Build without GUI (like CI)
make build-nogui

# Build with GUI (for local use)
make build
```

## Troubleshooting

### GUI Dependencies Missing
If you see errors like "X11/Xlib.h: No such file or directory":
- This is expected in CI/headless environments
- Use `make test` or `make build-nogui` instead
- Install GUI dependencies for local GUI development

### CGO Issues
If you see CGO-related compilation errors:
- Use `CGO_ENABLED=0` and `-tags nogui` flags
- This builds a headless version without GUI support
- Perfect for CI, Docker, and server deployments

## Release Builds

Release builds should include both versions:
- **GUI Version**: For desktop users (`make build`)
- **Headless Version**: For servers (`make build-nogui`)

This ensures compatibility across all deployment scenarios.