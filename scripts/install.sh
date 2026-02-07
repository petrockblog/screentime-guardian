#!/bin/bash
set -e

# Screentime Guardian Installation Script for Linux Mint
# Run as root: sudo ./install.sh

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/screentime-guardian"
DATA_DIR="/var/lib/screentime-guardian"
SERVICE_FILE="/etc/systemd/system/screentime-guardian.service"

echo "=== Screentime Guardian Installation ==="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo ./install.sh)"
    exit 1
fi

# Check for binary
BINARY="./screentime-guardian"
if [ ! -f "$BINARY" ]; then
    # Check in dist directory
    if [ -f "./dist/screentime-guardian-linux-amd64" ]; then
        BINARY="./dist/screentime-guardian-linux-amd64"
    else
        echo "Error: Binary not found. Please build first:"
        echo "  ./scripts/build.sh"
        exit 1
    fi
fi

echo "1. Installing binary..."
cp "$BINARY" "$INSTALL_DIR/screentime-guardian"
chmod 755 "$INSTALL_DIR/screentime-guardian"
echo "   Installed to $INSTALL_DIR/screentime-guardian"

echo ""
echo "2. Creating directories..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"
chmod 750 "$CONFIG_DIR"
chmod 750 "$DATA_DIR"

echo ""
echo "3. Creating configuration..."
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    cat > "$CONFIG_DIR/config.yaml" << 'EOF'
# Screentime Guardian Configuration

# Web interface address (use :8080 for all interfaces)
listen_addr: ":8080"

# Database location
database_path: "/var/lib/screentime-guardian/data.db"

# Admin password for web interface (set this!)
# Leave empty for first-run setup via web UI
admin_password: ""

# Warning intervals (minutes before lockout)
warning_intervals:
  - 5
  - 1

# How often to check time limits
check_interval: 30s

# Grace period after limit before hard lock
grace_period: 1m
EOF
    echo "   Created $CONFIG_DIR/config.yaml"
    echo "   ⚠️  Remember to set admin_password!"
else
    echo "   Config already exists, skipping"
fi

echo ""
echo "4. Installing systemd service..."
cp ./systemd/screentime-guardian.service "$SERVICE_FILE"
systemctl daemon-reload
echo "   Installed systemd service"

echo ""
echo "5. Installing dependencies..."
# Install avahi for mDNS
if command -v apt-get &> /dev/null; then
    apt-get install -y avahi-daemon libnotify-bin > /dev/null 2>&1 || true
    echo "   Installed avahi-daemon and libnotify"
fi

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo ""
echo "  1. Edit the configuration:"
echo "     sudo nano $CONFIG_DIR/config.yaml"
echo ""
echo "  2. Set a strong admin password in the config"
echo ""
echo "  3. Enable and start the service:"
echo "     sudo systemctl enable screentime-guardian"
echo "     sudo systemctl start screentime-guardian"
echo ""
echo "  4. Access the web interface:"
echo "     http://localhost:8080"
echo "     http://screentime-guardian.local:8080 (if mDNS works)"
echo ""
echo "  5. Add Linux users to track (they must exist as system users)"
echo ""
echo "To check status: sudo systemctl status screentime-guardian"
echo "To view logs:    sudo journalctl -u screentime-guardian -f"
