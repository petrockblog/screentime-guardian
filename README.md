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

### Step 1: Download the Package

Download the latest release for your architecture from the [releases page](https://github.com/petrockblog/screentime-guardian/releases):

```bash
# For AMD64 (Intel/AMD - most desktops and laptops)
wget https://github.com/petrockblog/screentime-guardian/releases/latest/download/screentime-guardian_*_amd64.deb

# For ARM64 (Raspberry Pi 4/5)
wget https://github.com/petrockblog/screentime-guardian/releases/latest/download/screentime-guardian_*_arm64.deb
```

### Step 2: Install the Package

Install with automatic dependency resolution:

```bash
sudo apt-get install -f ./screentime-guardian_*_amd64.deb
```

The installation will:
- Install the daemon to `/usr/bin/screentime-guardian`
- Create a default config at `/etc/screentime-guardian/config.yaml`
- Set up a systemd service
- Enable mDNS for easy access via `screentime-guardian.local`

### Step 3: Configure Admin Password

**⚠️ IMPORTANT**: The default installation has **no password** set. You must configure one before the service is secure.

Edit the configuration file:

```bash
sudo nano /etc/screentime-guardian/config.yaml
```

Set a strong admin password:

```yaml
listen_addr: ":8080"
database_path: "/var/lib/screentime-guardian/data.db"
admin_password: "YourSecurePasswordHere"  # ← CHANGE THIS!
warning_intervals:
  - 5
  - 1
check_interval: 30s
grace_period: 1m
```

### Step 4: Start the Service

```bash
sudo systemctl start screentime-guardian
sudo systemctl enable screentime-guardian  # Auto-start on boot
sudo systemctl status screentime-guardian   # Verify it's running
```

### Step 5: Access the Web Interface

Open your browser and go to:

- **From the same machine**: `http://localhost:8080`
- **From your phone/tablet**: `http://screentime-guardian.local:8080`
- **Using IP address**: `http://192.168.x.x:8080` (replace with your machine's IP)

**Login credentials**:
- **Username**: `admin`
- **Password**: The password you set in `/etc/screentime-guardian/config.yaml`

If you didn't set a password yet, the interface will be **unprotected** — set one immediately!

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

## Building from Source (Advanced)

If you need to build from source for development or modifications:

```bash
# Clone the repository
git clone https://github.com/petrockblog/screentime-guardian
cd screentime-guardian

# Install dependencies
go mod download

# Build for Linux
./scripts/build.sh

# Or build .deb packages (requires Linux with dpkg-dev, debhelper)
./scripts/build-deb.sh
```

**Development workflow**:

```bash
# Run locally (macOS - limited functionality, no D-Bus)
go run ./cmd/daemon -config configs/config.yaml.example

# Run tests
go test ./...
```

**Creating a release**: Push a git tag (e.g., `v1.0.1`) to trigger GitHub Actions, which automatically builds and publishes AMD64/ARM64 .deb packages.

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
