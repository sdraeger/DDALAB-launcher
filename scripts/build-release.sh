#!/bin/bash

# Build release versions for all platforms

set -e

BINARY_NAME="ddalab-launcher"
VERSION="1.0.0"
BUILD_DIR="dist"

echo "Building DDALAB Launcher v${VERSION}..."

# Create build directory
mkdir -p ${BUILD_DIR}

# Build for Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" \
    -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/launcher

# Build for macOS
echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" \
    -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ./cmd/launcher
GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=${VERSION}" \
    -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ./cmd/launcher

# Build for Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" \
    -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/launcher

# Create macOS app bundle
echo "Creating macOS app bundle..."
APP_NAME="DDALAB Launcher.app"
APP_DIR="${BUILD_DIR}/${APP_NAME}"
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

# Copy the appropriate binary based on current architecture
if [[ $(uname -m) == "arm64" ]]; then
    cp ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 "${APP_DIR}/Contents/MacOS/${BINARY_NAME}"
else
    cp ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 "${APP_DIR}/Contents/MacOS/${BINARY_NAME}"
fi

# Copy Info.plist
if [ -f "build/macos/Info.plist" ]; then
    cp build/macos/Info.plist "${APP_DIR}/Contents/"
fi

# Create PkgInfo
echo "APPL????" > "${APP_DIR}/Contents/PkgInfo"

# Make binaries executable
chmod +x ${BUILD_DIR}/${BINARY_NAME}-*
chmod +x "${APP_DIR}/Contents/MacOS/${BINARY_NAME}"

# Create archives
echo "Creating release archives..."
cd ${BUILD_DIR}

# Linux
tar czf ${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz ${BINARY_NAME}-linux-amd64

# macOS
tar czf ${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz ${BINARY_NAME}-darwin-amd64
tar czf ${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz ${BINARY_NAME}-darwin-arm64
zip -r ${BINARY_NAME}-${VERSION}-macos-app.zip "${APP_NAME}"

# Windows
zip ${BINARY_NAME}-${VERSION}-windows-amd64.zip ${BINARY_NAME}-windows-amd64.exe

cd ..

echo "Build complete! Release files are in ${BUILD_DIR}/"
ls -la ${BUILD_DIR}/