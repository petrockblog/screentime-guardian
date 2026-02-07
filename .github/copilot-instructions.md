# Screentime Guardian - AI Coding Instructions

## Architecture Overview

This is a Go daemon for Linux Mint that enforces per-user screen time limits. The application runs as a systemd service (root) and provides a mobile-friendly web UI for parents.

**Target environment**: Linux Mint with Cinnamon desktop environment (uses `cinnamon-screensaver` for locking, `notify-send` for notifications).

**Data flow**: Web UI → API handlers → Storage/Scheduler → D-Bus (logind) → Lock session

```
cmd/daemon/main.go          # Entry point, wires all components
internal/
├── api/                    # Chi router + htmx templates (embedded via //go:embed)
├── config/                 # YAML config loading
├── dbus/                   # systemd-logind session control + desktop notifications
├── mdns/                   # Zeroconf/Bonjour for screentime-guardian.local discovery
├── notifier/               # Interface for notifications (extensible for Telegram)
├── scheduler/              # Time tracking loop, warning triggers, lock enforcement
└── storage/                # SQLite via modernc.org/sqlite (pure Go, CGO_ENABLED=0)
```

## Key Patterns

### Notifier Interface (Extensibility)
New notification backends (e.g., Telegram) implement this interface in `internal/notifier/`:
```go
type Notifier interface {
    SendWarning(ctx context.Context, username string, minutesLeft int) error
    SendLockNotice(ctx context.Context, username string) error
    SendTimeExtended(ctx context.Context, username string, minutes int) error
}
```
Add to the `Chain` in `main.go` - all notifiers receive events.

### Embedded Web Assets
Templates and static files are embedded at compile time:
```go
//go:embed templates/*.html
var templatesFS embed.FS
```
Result: Single binary deployment, no external file dependencies.

### D-Bus Session Control
Session management uses `org.freedesktop.login1` (systemd-logind):
- `LockSession(id)` - locks screen (user must enter password)
- `TerminateSession(id)` - force logout
- `ListSessions()` - enumerate active users

Desktop notifications run via `runuser -u <username> -- notify-send` since daemon runs as root.

## Build & Development

```bash
# Build for all platforms (from macOS)
./scripts/build.sh          # Outputs to dist/

# Cross-compile manually
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/parental-control ./cmd/daemon

# Local testing (limited - D-Bus unavailable on macOS)
go run ./cmd/daemon -config configs/config.yaml.example
```

## Conventions

- **Error handling**: Wrap with `fmt.Errorf("context: %w", err)` for stack traces
- **Logging**: Use `log.Printf` (no structured logging yet)
- **Config**: YAML with defaults in `config.Default()` - graceful fallback if file missing
- **Database**: Schema migrations in `storage.migrate()` - add new tables there
- **Templates**: Pico CSS (classless) + htmx for interactivity, no JS build step

## Testing on Linux

The daemon requires Linux for full functionality (D-Bus, session locking). Deploy to test:
```bash
scp dist/parental-control-linux-amd64 user@linux-vm:~/
ssh user@linux-vm 'sudo ./parental-control -config ./config.yaml'
```

**Note**: No unit tests exist yet. When adding tests, mock the `dbus.LogindClient` and `notifier.Notifier` interfaces for isolation.

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

