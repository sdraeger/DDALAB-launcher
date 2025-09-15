# CI/CD Configuration

## Overview

The launcher uses GitHub Actions for continuous integration and testing. The configuration provides simple, reliable builds across all platforms.

## Workflows

### Test Workflow (`.github/workflows/test.yml`)

**Main Test Job (`test`):**
- Runs on Ubuntu 
- Tests core functionality
- Performs cross-platform build tests (Linux, macOS, Windows)
- Uses standard Go build process (no special tags required)

### Release Workflow (`.github/workflows/release.yml`)

**Test Job:**
- Same as test workflow
- Must pass before builds are created

**Build Job:**
- Creates cross-platform binaries
- Generates archives and checksums
- Creates GitHub releases with proper versioning

## CI Commands

The CI uses these standard commands:

```bash
# Tests
go test -v ./...

# Cross-platform builds
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/launcher
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./cmd/launcher  
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./cmd/launcher
```

## Local Development

Developers can use the same commands locally:

```bash
# Run tests like CI does
make test

# Build like CI does
make build
```

## Troubleshooting

### Build Issues
If you see compilation errors:
- Ensure Go version matches CI (1.21+)
- Check that dependencies are properly downloaded (`go mod download`)
- Verify code formatting (`make fmt`)

### Test Failures
If tests fail locally but should pass in CI:
- Run `go test -v ./...` to see detailed output
- Check for platform-specific issues
- Ensure no external dependencies are required

## Release Builds

Release builds are simple and reliable:
- **Cross-platform**: Works on Linux, macOS, Windows (both Intel and ARM)
- **Self-contained**: No external dependencies required
- **Lightweight**: Fast builds and small binaries

This ensures compatibility across all deployment scenarios.