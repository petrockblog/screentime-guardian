#!/bin/bash
set -e

# Build script for Screentime Guardian
# Cross-compiles from macOS to Linux

cd "$(dirname "$0")/.."

VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "=== Building Screentime Guardian ==="
echo "Version: $VERSION"
echo ""

mkdir -p dist

# Linux AMD64
echo "Building for Linux AMD64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o dist/screentime-guardian-linux-amd64 \
    ./cmd/daemon

# Linux ARM64 (for Raspberry Pi, etc.)
echo "Building for Linux ARM64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o dist/screentime-guardian-linux-arm64 \
    ./cmd/daemon

# macOS (for development/testing)
echo "Building for macOS..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o dist/screentime-guardian-darwin-arm64 \
    ./cmd/daemon

echo ""
echo "=== Build Complete ==="
echo ""
ls -lh dist/
echo ""
echo "To deploy to Linux Mint:"
echo "  scp dist/screentime-guardian-linux-amd64 user@target:~/"
echo "  scp -r systemd scripts user@target:~/"
echo "  ssh user@target 'sudo ~/scripts/install.sh'"
