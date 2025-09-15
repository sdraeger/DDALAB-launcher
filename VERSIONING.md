# Version Control Guide

This document explains how versioning works in the DDALAB Launcher repository and how to control version increments.

## Automatic Versioning

The DDALAB Launcher uses **semantic versioning** (semver) with automatic version management through GitHub Actions.

### Version Format: `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes or significant architectural changes
- **MINOR**: New features, enhancements, backward-compatible changes
- **PATCH**: Bug fixes, small improvements, documentation updates

## Version Increment Control

### 1. Default Behavior (Patch Increment)

**By default**, every push to the `main` branch increments the **patch version**:
- `v0.1.0` → `v0.1.1`
- `v0.1.5` → `v0.1.6`

### 2. Commit Message Keywords

Control version increments using specific keywords in your commit messages:

#### Major Version Increment (`MAJOR`)
Use any of these patterns in your commit message:
```bash
git commit -m "[major] Redesign CLI interface with breaking changes"
git commit -m "[breaking] Remove deprecated commands"
git commit -m "major: Complete rewrite of core functionality"
git commit -m "breaking: Change configuration file format"
```

**Result**: `v0.1.5` → `v1.0.0`

#### Minor Version Increment (`MINOR`)
Use any of these patterns in your commit message:
```bash
git commit -m "[minor] Add new export functionality"
git commit -m "[feature] Implement real-time status updates"
git commit -m "[feat] Add support for ARM64 platforms"
git commit -m "minor: Add configuration validation"
git commit -m "feature: Implement backup scheduling"
git commit -m "feat: Add Docker health checks"
```

**Result**: `v0.1.5` → `v0.2.0`

#### Patch Version Increment (`PATCH`) - Default
Any commit message without the above keywords:
```bash
git commit -m "Fix typo in error message"
git commit -m "Update dependencies"
git commit -m "Improve error handling"
```

**Result**: `v0.1.5` → `v0.1.6`

### 3. Manual Workflow Dispatch

You can manually trigger a release with specific version increment:

1. Go to **GitHub Actions** → **"Build and Release"**
2. Click **"Run workflow"**
3. Select branch: `main`
4. Choose version type:
   - `patch` - Bug fixes (0.1.0 → 0.1.1)
   - `minor` - New features (0.1.0 → 0.2.0)
   - `major` - Breaking changes (0.1.0 → 1.0.0)

## Examples

### Typical Development Flow

```bash
# Bug fix - patch increment
git commit -m "Fix memory leak in status checker"
git push origin main
# Creates: v0.1.1

# New feature - minor increment
git commit -m "[feature] Add configuration backup"
git push origin main
# Creates: v0.2.0

# Another bug fix - patch increment
git commit -m "Fix error handling in backup command"
git push origin main
# Creates: v0.2.1

# Breaking change - major increment
git commit -m "[breaking] Change command structure and remove deprecated flags"
git push origin main
# Creates: v1.0.0
```

### Keyword Patterns Supported

| Type | Keywords | Pattern Examples |
|------|----------|------------------|
| **Major** | `major`, `breaking` | `[major]`, `[breaking]`, `major:`, `breaking:` |
| **Minor** | `minor`, `feature`, `feat` | `[minor]`, `[feature]`, `[feat]`, `minor:`, `feature:`, `feat:` |
| **Patch** | *any other* | Default for all other commit messages |

## Release Artifacts

Each release automatically creates:

### Binaries for All Platforms
- **Linux**: `ddalab-launcher-linux-amd64.tar.gz`, `ddalab-launcher-linux-arm64.tar.gz`
- **macOS**: `ddalab-launcher-darwin-amd64.tar.gz`, `ddalab-launcher-darwin-arm64.tar.gz`
- **Windows**: `ddalab-launcher-windows-amd64.zip`, `ddalab-launcher-windows-arm64.zip`

### Additional Files
- `checksums.txt` - SHA256 checksums for verification
- Release notes with installation instructions

## Best Practices

### 1. Use Descriptive Commit Messages
```bash
# Good
git commit -m "[feature] Add real-time log streaming with WebSocket support"
git commit -m "Fix race condition in interrupt handler"

# Avoid
git commit -m "[feature] stuff"
git commit -m "fix"
```

### 2. Plan Your Version Increments
- **Patch**: Bug fixes, documentation, small improvements
- **Minor**: New features, enhancements that don't break existing functionality
- **Major**: Breaking changes, API changes, architectural redesigns

### 3. Test Before Releasing
The workflow includes automatic testing, but ensure your changes work locally:
```bash
# Run tests
go test ./...

# Build for your platform
go build cmd/launcher/main.go

# Test the binary
./main --version
```

### 4. Review Release Notes
Each release includes auto-generated notes. The workflow will show:
- Which increment type was used
- Platform-specific download instructions
- Verification instructions

## Troubleshooting

### Issue: Wrong Version Increment
If the wrong version was released, you can:
1. Delete the incorrect tag/release on GitHub
2. Make a new commit with the correct keyword
3. Push to trigger a new release

### Issue: Need to Skip Release
To push without creating a release, use the test workflow instead or create a feature branch:
```bash
git checkout -b fix/some-issue
git commit -m "Work in progress fix"
git push origin fix/some-issue
# No release created, only tests run
```

## Monitoring

You can monitor the versioning process by:
1. Checking the **Actions** tab for workflow runs
2. Reviewing the workflow logs to see detected increment types
3. Checking the **Releases** page for generated releases

The workflow logs will show:
```
Latest tag: v0.1.0
Commit message: [feature] Add new export functionality
Detected MINOR version increment from commit message
Increment type: minor
New version: 0.2.0
New tag: v0.2.0
```