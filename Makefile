.PHONY: build run clean test install deps fmt lint vet pre-commit pre-commit-quick check-fmt setup-hooks

# Binary name
BINARY_NAME=ddalab-launcher

# Build directory
BUILD_DIR=bin

# Go build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: deps build

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build the launcher
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/launcher

# Build for multiple platforms
build-all: deps
	mkdir -p $(BUILD_DIR)
	# Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/launcher
	# macOS
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/launcher
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/launcher
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/launcher

# Build macOS app bundle
build-macos-app: build
	mkdir -p $(BUILD_DIR)/DDALAB\ Launcher.app/Contents/MacOS
	mkdir -p $(BUILD_DIR)/DDALAB\ Launcher.app/Contents/Resources
	cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/DDALAB\ Launcher.app/Contents/MacOS/
	cp build/macos/Info.plist $(BUILD_DIR)/DDALAB\ Launcher.app/Contents/
	# Create a simple icon if needed
	echo "APPL????" > $(BUILD_DIR)/DDALAB\ Launcher.app/Contents/PkgInfo

# Run the launcher
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# Install to system (requires sudo on Unix systems)
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

# Development mode - rebuild and run on file changes
dev:
	go run ./cmd/launcher

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Code formatted successfully!"

# Lint code
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "Linting completed!"; \
	else \
		echo "golangci-lint not found locally. Install it or run in CI for linting."; \
		echo "To install: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
	fi

# Vet code
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet completed!"

# Pre-commit checks - run before committing
pre-commit: fmt vet lint test
	@echo "Pre-commit checks completed successfully!"

# Quick pre-commit checks (without golangci-lint)
pre-commit-quick: fmt vet test
	@echo "Quick pre-commit checks completed successfully!"

# Check if code is properly formatted
check-fmt:
	@echo "Checking code formatting..."
	@if [ "$$(gofmt -l . | wc -l)" -gt 0 ]; then \
		echo "Code is not properly formatted. Please run 'make fmt'"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "Code formatting is correct!"

# Setup git hooks for development
setup-hooks:
	@echo "Setting up Git hooks..."
	./scripts/setup-git-hooks.sh
	@echo "Git hooks setup completed!"
