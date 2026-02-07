#!/bin/bash
set -e

# Build .deb packages for screentime-guardian
# Requires: debhelper, dpkg-dev, golang

cd "$(dirname "$0")/.."

echo "=== Building Debian Packages ==="
echo ""

# Check for required tools
if ! command -v dpkg-buildpackage &> /dev/null; then
    echo "Error: dpkg-dev not installed. Run: sudo apt-get install dpkg-dev debhelper dh-golang"
    exit 1
fi

# Get version from git tags
VERSION=$(git describe --tags --always 2>/dev/null || echo "1.0.0")
echo "Version: $VERSION"
echo ""

# Clean previous builds
rm -rf debian/.debhelper debian/screentime-guardian debian/*.debhelper* debian/files
rm -f ../screentime-guardian_*.deb ../screentime-guardian_*.build* ../screentime-guardian_*.changes

# Build AMD64 package
echo "Building AMD64 package..."
dpkg-buildpackage -us -uc -b -aamd64
mv ../screentime-guardian_*.deb dist/ 2>/dev/null || true

# Clean for next build
rm -rf debian/.debhelper debian/screentime-guardian debian/*.debhelper* debian/files

# Build ARM64 package  
echo ""
echo "Building ARM64 package..."
dpkg-buildpackage -us -uc -b -aarm64
mv ../screentime-guardian_*.deb dist/ 2>/dev/null || true

# Clean up
rm -rf debian/.debhelper debian/screentime-guardian debian/*.debhelper* debian/files
rm -f ../screentime-guardian_*.build* ../screentime-guardian_*.changes

echo ""
echo "=== Build Complete ==="
ls -lh dist/screentime-guardian_*.deb 2>/dev/null || echo "Packages moved to dist/"
echo ""
echo "Test installation with:"
echo "  sudo apt-get install -f ./dist/screentime-guardian_*_amd64.deb"
