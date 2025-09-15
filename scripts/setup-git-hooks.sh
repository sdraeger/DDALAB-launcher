#!/bin/bash

# Setup Git Hooks for DDALAB Launcher
# This script installs pre-commit hooks to ensure code quality

set -e

echo "🔧 Setting up Git hooks for DDALAB Launcher..."

# Get the git directory (handles both normal repos and submodules)
GIT_DIR=$(git rev-parse --git-dir)
HOOKS_DIR="$GIT_DIR/hooks"

# Ensure hooks directory exists
mkdir -p "$HOOKS_DIR"

# Get the current script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Check if we already have a pre-commit hook
if [ -f "$HOOKS_DIR/pre-commit" ]; then
    echo "⚠️  Pre-commit hook already exists. Creating backup..."
    cp "$HOOKS_DIR/pre-commit" "$HOOKS_DIR/pre-commit.backup.$(date +%Y%m%d%H%M%S)"
fi

# Create the pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash

# DDALAB Launcher Pre-commit Hook
# This script runs formatting and linting checks before allowing commits

set -e

echo "🔍 Running pre-commit checks..."

# Change to the launcher directory
cd "$GIT_PREFIX" || cd "$(git rev-parse --show-toplevel)"

# Check if we have a Makefile
if [ ! -f "Makefile" ]; then
    echo "❌ No Makefile found. Cannot run pre-commit checks."
    exit 1
fi

# Format code automatically
echo "📝 Formatting Go code..."
make fmt

# Check if formatting changed any files
if ! git diff --exit-code --quiet; then
    echo "🔧 Code was automatically formatted. Please review and add the changes:"
    echo
    git diff --name-only
    echo
    echo "Run the following commands to add the formatting changes:"
    echo "  git add ."
    echo "  git commit"
    exit 1
fi

# Run code quality checks
echo "🔍 Running code quality checks..."

# Run go vet
echo "  → Running go vet..."
if ! make vet; then
    echo "❌ go vet failed. Please fix the issues and try again."
    exit 1
fi

# Check formatting (redundant but good practice)
echo "  → Checking code formatting..."
if ! make check-fmt; then
    echo "❌ Code formatting check failed. Please run 'make fmt' and try again."
    exit 1
fi

# Run linter if available
echo "  → Running linter..."
if command -v golangci-lint >/dev/null 2>&1; then
    if ! make lint; then
        echo "❌ Linting failed. Please fix the issues and try again."
        exit 1
    fi
else
    echo "⚠️  golangci-lint not found locally. Skipping lint check."
    echo "   Install it with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin"
fi

# Run tests to ensure nothing is broken
echo "  → Running tests..."
if ! make test; then
    echo "❌ Tests failed. Please fix the failing tests and try again."
    exit 1
fi

echo "✅ All pre-commit checks passed!"
echo
EOF

# Make the hook executable
chmod +x "$HOOKS_DIR/pre-commit"

echo "✅ Pre-commit hook installed successfully!"
echo
echo "The hook will now run automatically before each commit and will:"
echo "  • Format Go code automatically"
echo "  • Run go vet for code quality checks"
echo "  • Run golangci-lint (if available)"
echo "  • Run all tests"
echo
echo "To bypass the hook for emergency commits, use:"
echo "  git commit --no-verify"
echo
echo "To manually run the pre-commit checks:"
echo "  make pre-commit"
echo