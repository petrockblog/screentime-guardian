# Screentime Guardian

A screen time management application for Linux Mint that allows parents to set daily time limits for children's computer usage. Parents can extend time from any device on the home network via a mobile-friendly web interface.

## Features

- **Per-user time limits**: Set different daily limits for each child
- **On-screen warnings**: Children see countdown notifications before lockout
- **Web-based control**: Mobile-friendly interface accessible from any device
- **Time extensions**: Easily grant extra time with one tap
- **Session locking**: Automatically locks screen when time expires
- **Usage tracking**: View daily usage history for each user
- **mDNS discovery**: Access at `http://screentime-guardian.local:8080`

## Requirements

- Linux Mint (or any systemd-based Linux with MATE/Cinnamon/Xfce)
- Root access for installation
- Go 1.21+ (for building from source)

## Installation

### From Debian Package (Recommended)

Download the latest release for your architecture:

```bash
# For AMD64 (Intel/AMD processors)
wget https://github.com/florian/screentime-guardian/releases/download/v1.0.0/screentime-guardian_1.0.0-1_amd64.deb

# For ARM64 (Raspberry Pi 4/5)
wget https://github.com/florian/screentime-guardian/releases/download/v1.0.0/screentime-guardian_1.0.0-1_arm64.deb
```

Install with automatic dependency resolution:

```# Manual Installation (on Linux Mint)

```bash
# Copy files to target machine
scp -r dist/parental-control-linux-amd64 systemd scripts user@mint-pc:~/

# SSH to the machine and install
ssh user@mint-pc
sudo ./scripts/install.sh
```

**Note**: The Debian package installation method is preferred as it handles dependencies automatically.
Configure the daemon:

```bash
sudo nano /etc/screentime-guardian/config.yaml
# Set a secure admin_password!
```

Start the service:

```bash
sudo systemctl status screentime-guardian
sudo systemctl status screentime-guardian
```

Access the web interface at `http://localhost:8080` or `http://screentime-guardian.local:8080`

### From Source

For development or if you need to build from source:

#### Building (on macOS or Linux)

```bash
# Clone the repository
git clone https://github.com/florian/screentime-guardian
cd parental-control

# Install dependencies
go mod tidy

# Build for Linux
./scripts/build.sh
```

### Installation (on Linux Mint)

```bash
# Copy files to target machine
scp -r dist/parental-control-linux-amd64 systemd scripts user@mint-pc:~/

# SSH to the machine and install
ssh user@mint-pc
sudo ./scripts/install.sh
```

### Configuration

Edit `/etc/screentime-guardian/config.yaml`:

```yaml
listen_addr: ":8080"
database_path: "/var/lib/screentime-guardian/data.db"
admin_password: "your-secure-password"  # CHANGE THIS!
warning_intervals:
  - 5
  - 1
```

### Start the Service

```bash
sudo systemctl status screentime-guardian
sudo systemctl status screentime-guardian
```

### Access the Web Interface

- From the same machine: `http://localhost:8080`
- From your phone/tablet: `http://screentime-guardian.local:8080`
- Or use the IP address: `http://192.168.x.x:8080`

## Usage

1. **Add Users**: Go to Users → Add the Linux username of each child
2. **Set Limits**: Configure daily time limits (default: 120 minutes)
3. **Monitor**: Dashboard shows real-time status of all users
4. **Extend Time**: Tap the +15/+30/+60 min buttons when needed
5. **Lock Now**: Manually lock a child's screen if needed

## How It Works

The daemon runs as a systemd service and:

1. Tracks active user sessions via D-Bus (systemd-logind)
2. Counts screen time while users are logged in
3. Sends desktop notifications when time is running low
4. Locks the session using `loginctl lock-session` when time expires

Children see warnings at 5 minutes and 1 minute before lockout, giving them time to save their work.

## Security

- The daemon runs as root to manage user sessions
- Web interface is protected by HTTP Basic Auth
- Config file has restricted permissions (root-only)
- Children cannot stop or modify the service

## Architecture

```
┌─────────────────────────────────────────┐
│           Web Interface (Chi)           │
│         screentime-guardian.local          │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│              Scheduler                  │
│    (time tracking, enforcement)         │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│            D-Bus Clients                │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │   logind    │  │  Notifications   │  │
│  │  (sessions) │  │   (warnings)     │  │
│  └─────────────┘  └──────────────────┘  │
└─────────────────────────────────────────┘
```

## Extending with Telegram (Future)

The architecture includes a `Notifier` interface that allows adding Telegram notifications:

```go
type Notifier interface {
    SendWarning(ctx context.Context, username string, minutesLeft int) error
    SendLockNotice(ctx context.Context, username string) error
    SendTimeExtended(ctx context.Context, username string, minutes int) error
}
```

To add Telegram support, implement this interface and add it to the notifier chain.

## Development

```bash
# Run locally (macOS - limited functionality)
go run ./cmd/daemon -config configs/config.yaml.example

# Run tests
go test ./...

# Build all platforms
./scripts/build.sh

# Build Debian packages (requires Linux with dpkg-dev, debhelper, dh-golang)
./scripts/build-deb.sh
```

### Creating a Release

To create a new release with automated .deb packages:

```bash
# Commit all changes
git add .
git commit -m "Release version 1.0.0"

# Create and push tag (triggers GitHub Actions)
git tag v1.0.0
git push origin main
git push origin v1.0.0
```

GitHub Actions will automatically build AMD64 and ARM64 .deb packages and attach them to the release.

## Troubleshooting

### Service won't start
```bash
sudo journalctl -u screentime-guardian -f
```

### Can't access web interface
- Check firewall: `sudo ufw allow 8080/tcp`
- Verify service is running: `sudo systemctl status screentime-guardian`

### mDNS not working
- Install avahi: `sudo apt install avahi-daemon`
- Ensure avahi is running: `sudo systemctl status avahi-daemon`

### Notifications not showing
- The daemon uses `runuser` to send notifications as the target user
- Ensure `libnotify-bin` is installed: `sudo apt install libnotify-bin`

## License

MIT License
