.PHONY: build run clean test install deps

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
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/launcher

# Build for multiple platforms
build-all: deps
	mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/launcher
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/launcher
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/launcher
	# Windows (without console window)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -H=windowsgui -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/launcher

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
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Vet code
vet:
	go vet ./...