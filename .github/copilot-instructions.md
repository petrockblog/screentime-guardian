# Screentime Guardian - AI Coding Instructions

## Architecture Overview

This is a Go daemon for Linux Mint that enforces per-user screen time limits. The application runs as a systemd service (root) and provides a mobile-friendly web UI for parents.

**Target environment**: Linux Mint with Cinnamon/MATE/Xfce desktop environments (uses systemd-logind for session control, desktop notifications via `notify-send`).

**Data flow**: Web UI (htmx) → Chi API handlers → Storage/Scheduler → D-Bus (logind) → Lock/Terminate session

```
cmd/daemon/main.go          # Entry point, wires all components
internal/
├── api/                    # Chi router + htmx templates (embedded via //go:embed)
│   ├── handlers.go         # HTTP endpoint logic
│   ├── router.go           # Route definitions + middleware
│   └── templates/*.html    # Pico CSS (classless) + htmx for interactivity
├── config/                 # YAML config loading with defaults
├── dbus/                   # systemd-logind session control + desktop notifications
│   ├── logind.go           # Session locking/termination via D-Bus
│   ├── notify.go           # Desktop notifications (runs as target user)
│   └── mock.go             # Mock implementations for testing on macOS
├── mdns/                   # Zeroconf/Bonjour for screentime-guardian.local discovery
├── notifier/               # Extensible notification chain pattern (desktop + future Telegram)
├── scheduler/              # Time tracking loop, warning triggers, lock enforcement
└── storage/                # SQLite via modernc.org/sqlite (pure Go, CGO_ENABLED=0)
```

## Key Patterns

### Notifier Chain (Extensibility)
New notification backends (e.g., Telegram) implement the `Notifier` interface in [internal/notifier/](internal/notifier/):
```go
type Notifier interface {
    SendWarning(ctx context.Context, username string, minutesLeft int) error
    SendLockNotice(ctx context.Context, username string) error
    SendTimeExtended(ctx context.Context, username string, minutes int) error
}
```
**Chain pattern**: Add to `notifier.NewChain()` in [cmd/daemon/main.go](cmd/daemon/main.go) - all notifiers receive events sequentially.

### Embedded Web Assets
Templates and static files are compiled into the binary using `//go:embed` in [internal/api/router.go](internal/api/router.go):
```go
//go:embed templates/*.html
var templatesFS embed.FS
```
**Result**: Single binary deployment, no external file dependencies. Templates use Pico CSS (classless) + htmx for interactivity, no JS build step.

### D-Bus Session Control
Session management via `org.freedesktop.login1` (systemd-logind) in [internal/dbus/logind.go](internal/dbus/logind.go):
- `LockSession(id)` - locks screen (user must enter password to unlock)
- `TerminateSession(id)` - force logout (used as fallback when lock unavailable)
- `ListSessions()` - enumerate active user sessions

**Critical**: Desktop notifications run via `runuser -u <username> -- notify-send` since daemon runs as root and must target user's session bus.

### Mock Pattern for Cross-Platform Development
Since D-Bus is Linux-only, [internal/dbus/mock.go](internal/dbus/mock.go) provides test implementations:
- `MockLogindClient` - simulates session operations
- `MockNotifier` - simulates desktop notifications

Used in tests (see [internal/scheduler/scheduler_test.go](internal/scheduler/scheduler_test.go)) to verify logic on macOS during development.

## Build & Development

### Local Development (on macOS/Linux)
```bash
# Run with example config (limited - D-Bus unavailable on macOS)
go run ./cmd/daemon -config configs/config.yaml.example

# Run tests (mocks enable testing on any platform)
go test ./...
go test -v ./internal/scheduler  # See time tracking logic
go test -cover ./...              # Check coverage

# Cross-compile for Linux (CGO disabled for pure Go SQLite)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/screentime-guardian-linux-amd64 ./cmd/daemon
```

### Build All Platforms
```bash
./scripts/build.sh  # Outputs to dist/ (Linux AMD64, ARM64, macOS ARM64)
```

### Debian Packaging (Linux only)
```bash
# Prerequisites
sudo apt-get install debhelper devscripts dpkg-dev dh-golang

# Build .deb packages
./scripts/build-deb.sh  # Creates AMD64 and ARM64 packages in dist/

# Test installation
sudo apt-get install -f ./dist/screentime-guardian_*.deb
systemctl status screentime-guardian
```

### Deploy & Test on Linux
Full functionality (D-Bus, session locking) requires Linux. Deploy manually:
```bash
scp dist/screentime-guardian-linux-amd64 user@linux-vm:~/
ssh user@linux-vm 'sudo ./screentime-guardian -config ./config.yaml'
```

### Release Workflow
GitHub Actions automates releases ([.github/workflows/release.yml](.github/workflows/release.yml)):
1. Tag version: `git tag v1.0.0 && git push origin v1.0.0`
2. Actions builds AMD64/ARM64 .deb packages
3. Uploads to GitHub Releases with SHA256 checksums

See [PACKAGING.md](PACKAGING.md) for detailed packaging documentation.

## Conventions

- **Error handling**: Wrap with `fmt.Errorf("context: %w", err)` for stack traces
- **Logging**: Use `log.Printf` (no structured logging yet)
- **Config**: YAML with defaults in `config.Default()` - graceful fallback if file missing
- **Database**: 
  - SQLite with WAL mode + foreign keys enabled (see [internal/storage/storage.go](internal/storage/storage.go))
  - Schema migrations in `storage.migrate()` - add new tables there
  - Uses `modernc.org/sqlite` (pure Go, not `mattn/go-sqlite3`) for CGO_ENABLED=0 cross-compilation
- **Templates**: 
  - Pico CSS (classless) + htmx (v1.9.10 from CDN) for interactivity
  - No JS build step or framework - server-rendered with htmx attributes (`hx-post`, `hx-swap`, etc.)
  - Template functions defined in `router.go` funcMap (e.g., `divf` for float division)
- **Testing**:
  - Test files: `*_test.go` in same package as code
  - Mocks: Separate `mock.go` files (e.g., `dbus/mock.go`, `notifier/mock.go`)
  - See [internal/README_TESTS.md](internal/README_TESTS.md) for coverage goals and mock strategy
  - **API testing**: Future tests should use `net/http/httptest` to test handlers:
    ```go
    req := httptest.NewRequest("POST", "/api/users", body)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    ```
- **Scheduler warnings**:
  - Configured via `warning_intervals` (default: `[5, 1]` minutes before lockout)
  - Warnings tracked per-user in `scheduler.warningsSent` map to prevent duplicates
  - Reset daily at midnight or when time is extended via `scheduler.ResetWarnings(username)`
  - Check interval defaults to 30 seconds (`check_interval` in config)

## Configuration

Config file: `/etc/screentime-guardian/config.yaml` (see [configs/config.yaml.example](configs/config.yaml.example))

**Required fields**:
- `admin_password` - Web UI password (must be set on first run)

**Optional fields with defaults**:
- `listen_addr` - Web UI address (default: `:8080`)
- `database_path` - SQLite database location (default: `/var/lib/screentime-guardian/data.db`)
- `enable_tls` - Enable HTTPS (default: `false`)
- `tls_cert_file` - Path to TLS certificate (default: `/etc/screentime-guardian/server.crt`)
- `tls_key_file` - Path to TLS private key (default: `/etc/screentime-guardian/server.key`)
- `warning_intervals` - Minutes before lockout to send warnings (default: `[5, 1]`)
- `check_interval` - Time between scheduler checks (default: `30s`)
- `grace_period` - Extra time after limit before hard lock (default: `1m`)

**Example**:
```yaml
listen_addr: ":8080"
database_path: "/var/lib/screentime-guardian/data.db"
admin_password: "your-secure-password"
enable_tls: true
tls_cert_file: "/etc/screentime-guardian/server.crt"
tls_key_file: "/etc/screentime-guardian/server.key"
warning_intervals:
  - 15  # Warn at 15 minutes
  - 5   # Warn at 5 minutes  
  - 1   # Final warning at 1 minute
check_interval: 30s
grace_period: 2m
```

**HTTPS Setup**:
Generate self-signed certificate with:
```bash
sudo ./scripts/generate-cert.sh
# Then set enable_tls: true in config.yaml
```

## Testing on Linux

The daemon requires Linux for full functionality (D-Bus, session locking). Deploy to test:
```bash
# Deploy manually
scp dist/screentime-guardian-linux-amd64 user@linux-vm:~/
ssh user@linux-vm 'sudo ./screentime-guardian -config ./config.yaml'

# Or test with .deb package
scp dist/screentime-guardian_*.deb user@linux-vm:~/
ssh user@linux-vm 'sudo apt-get install -f ./screentime-guardian_*.deb'
```

**Testing strategy**: Unit tests use mock implementations (`dbus/mock.go`, `notifier/mock.go`) to isolate logic from Linux-specific D-Bus dependencies. This enables test execution on macOS during development.

## Future: Telegram Integration

To add Telegram notifications for parents:

1. Create `internal/notifier/telegram.go` implementing the `Notifier` interface
2. Use `github.com/go-telegram-bot-api/telegram-bot-api/v5` for the bot API
3. Add config fields: `telegram_bot_token`, `telegram_chat_ids` in `config.Config`
4. Register in `main.go`:
   ```go
   if cfg.TelegramBotToken != "" {
       tgNotifier := notifier.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatIDs)
       notifierChain.Add(tgNotifier)
   }
   ```
5. Parents link via `/start` command; store chat IDs in SQLite `settings` table

## Gotchas

- **Pure Go SQLite**: Uses `modernc.org/sqlite` (not `mattn/go-sqlite3`) for CGO-free cross-compilation
- **Notification target**: Notifications go to user's session bus, not system bus - requires `runuser`
- **No unlock API**: `logind` has no unlock; users must enter password. Extend time instead.
- **Version injection**: Use ldflags in build scripts: `-ldflags "-X main.Version=${VERSION}"` for version display
