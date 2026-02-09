# Debian Packaging Guide

This document explains the Debian packaging setup for screentime-guardian.

## Files Created

### Debian Package Metadata (`debian/`)

- **control**: Package dependencies, description, maintainer info
- **changelog**: Version history (Debian format)
- **compat**: debhelper compatibility level (13)
- **copyright**: License information (MIT)
- **source/format**: Source package format (3.0 native)
- **rules**: Build instructions (Makefile format)
- **dirs**: Directories to create during installation
- **conffiles**: Configuration files protected from overwrites
- **postinst**: Post-installation script (creates config, displays setup message)
- **prerm**: Pre-removal script (stops service)

### Build Scripts

- **scripts/build-deb.sh**: Local build script for creating .deb packages
- **.github/workflows/release.yml**: Automated release workflow for GitHub Actions

### Documentation

- **README.md**: Updated with Debian package installation instructions
- **.gitignore**: Added Debian build artifacts

## Building Packages Locally

### Prerequisites (on Linux)

```bash
sudo apt-get install debhelper devscripts dpkg-dev dh-golang golang-go
```

### Build Process

```bash
# Build both AMD64 and ARM64 packages
./scripts/build-deb.sh

# Output files in dist/:
#   screentime-guardian_1.0.0-1_amd64.deb
#   screentime-guardian_1.0.0-1_arm64.deb
```

### Test Installation

```bash
# Install with dependency resolution
sudo apt-get install -f ./dist/screentime-guardian_1.0.0-1_amd64.deb

# Verify installation
systemctl status screentime-guardian

# Check files
ls -la /usr/local/bin/screentime-guardian
ls -la /etc/screentime-guardian/config.yaml
ls -la /lib/systemd/system/screentime-guardian.service
```

## Creating a Release on GitHub

### 1. Prepare the Release

Update the version in `debian/changelog`:

```bash
dch -v 1.0.0-1
# Edit the changelog entry, save, and commit
```

### 2. Create and Push Tag

```bash
git add debian/changelog
git commit -m "Prepare release v1.0.0"
git tag v1.0.0
git push origin main
git push origin v1.0.0
```

### 3. Automated Build

GitHub Actions will automatically:
1. Build AMD64 package
2. Build ARM64 package
3. Generate SHA256 checksums
4. Create GitHub Release
5. Upload all artifacts

### 4. Release Assets

The release will contain:
- `screentime-guardian_1.0.0-1_amd64.deb` (Intel/AMD)
- `screentime-guardian_1.0.0-1_arm64.deb` (Raspberry Pi)
- `SHA256SUMS` (verification)

## User Installation

Users can install directly from GitHub releases:

```bash
# Download
wget https://github.com/florian/screentime-guardian/releases/download/v1.0.0/screentime-guardian_1.0.0-1_amd64.deb

# Install with dependencies
sudo apt-get install -f ./screentime-guardian_1.0.0-1_amd64.deb

# Configure
sudo nano /etc/screentime-guardian/config.yaml

# Start
sudo systemctl status screentime-guardian
```

## Package Details

### Dependencies

The package automatically installs:
- systemd (process management)
- dbus (IPC for session control)
- avahi-daemon (mDNS discovery)
- libnotify-bin (desktop notifications)

### File Locations

- Binary: `/usr/local/bin/screentime-guardian`
- Config: `/etc/screentime-guardian/config.yaml` (conffile - protected)
- Database: `/var/lib/screentime-guardian/data.db` (created at runtime)
- Service: `/lib/systemd/system/screentime-guardian.service`
- Examples: `/usr/share/doc/screentime-guardian/examples/config.yaml`

### Systemd Integration

The package uses `dh-systemd` for automatic:
- Service file installation
- `systemctl daemon-reload` after install
- Service start on install (if configured)
- Service stop on removal

## Versioning

Use semantic versioning with Debian revision:
- `1.0.0-1`: Version 1.0.0, Debian revision 1
- `1.0.1-1`: Version 1.0.1, Debian revision 1
- `1.0.1-2`: Version 1.0.1, Debian revision 2 (packaging fix)

Git tags should use `v` prefix: `v1.0.0`, `v1.0.1`, etc.

## Future Enhancements

### Package Signing

To add GPG signing for package authenticity:

```bash
# Generate GPG key
gpg --gen-key

# Sign packages
dpkg-buildpackage -k<KEYID>

# Users import public key
gpg --keyserver keyserver.ubuntu.com --recv-keys <KEYID>
```

### PPA Distribution

For `apt install screentime-guardian` support:

1. Create Launchpad account
2. Upload signed source package
3. Launchpad builds for all Ubuntu versions
4. Users add PPA: `sudo add-apt-repository ppa:username/screentime-guardian`

### Architecture Support

Current: AMD64, ARM64  
Potential: ARMv7 (older Raspberry Pi), i386 (legacy)

Add to `debian/control` Architecture field and build script.
