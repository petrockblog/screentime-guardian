package notifier

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"

	"github.com/petrockblog/screentime-guardian/internal/dbus"
)

// Notifier is the interface for sending notifications to users
type Notifier interface {
	SendWarning(ctx context.Context, username string, minutesLeft int) error
	SendLockNotice(ctx context.Context, username string) error
	SendTimeExtended(ctx context.Context, username string, minutes int) error
}

// Chain combines multiple notifiers, sending to all of them
type Chain struct {
	notifiers []Notifier
}

// NewChain creates a notifier chain
func NewChain(notifiers ...Notifier) *Chain {
	return &Chain{notifiers: notifiers}
}

// Add adds a notifier to the chain
func (c *Chain) Add(n Notifier) {
	c.notifiers = append(c.notifiers, n)
}

// SendWarning sends a warning through all notifiers
func (c *Chain) SendWarning(ctx context.Context, username string, minutesLeft int) error {
	var lastErr error
	for _, n := range c.notifiers {
		if err := n.SendWarning(ctx, username, minutesLeft); err != nil {
			log.Printf("Warning notification failed: %v", err)
			lastErr = err
		}
	}
	return lastErr
}

// SendLockNotice sends a lock notice through all notifiers
func (c *Chain) SendLockNotice(ctx context.Context, username string) error {
	var lastErr error
	for _, n := range c.notifiers {
		if err := n.SendLockNotice(ctx, username); err != nil {
			log.Printf("Lock notification failed: %v", err)
			lastErr = err
		}
	}
	return lastErr
}

// SendTimeExtended sends a time extension notice through all notifiers
func (c *Chain) SendTimeExtended(ctx context.Context, username string, minutes int) error {
	var lastErr error
	for _, n := range c.notifiers {
		if err := n.SendTimeExtended(ctx, username, minutes); err != nil {
			log.Printf("Extension notification failed: %v", err)
			lastErr = err
		}
	}
	return lastErr
}

// DBusNotifier sends desktop notifications via D-Bus
type DBusNotifier struct {
	notifier *dbus.Notifier
}

// NewDBusNotifier creates a new D-Bus based notifier
func NewDBusNotifier(n *dbus.Notifier) *DBusNotifier {
	return &DBusNotifier{notifier: n}
}

// SendWarning sends a desktop warning notification
func (d *DBusNotifier) SendWarning(ctx context.Context, username string, minutesLeft int) error {
	return sendNotifyAsUser(username, "Time Warning",
		fmt.Sprintf("You have %d minute(s) of screen time remaining.", minutesLeft),
		getUrgency(minutesLeft))
}

// SendLockNotice sends a desktop lock notification
func (d *DBusNotifier) SendLockNotice(ctx context.Context, username string) error {
	return sendNotifyAsUser(username, "Time's Up!",
		"Your screen time has ended. The session will now be locked.",
		"critical")
}

// SendTimeExtended sends a time extension notification
func (d *DBusNotifier) SendTimeExtended(ctx context.Context, username string, minutes int) error {
	return sendNotifyAsUser(username, "Time Extended",
		fmt.Sprintf("Your screen time has been extended by %d minutes.", minutes),
		"normal")
}

func getUrgency(minutesLeft int) string {
	if minutesLeft <= 1 {
		return "critical"
	}
	return "normal"
}

func sendNotifyAsUser(username, summary, body, urgency string) error {
	notifyCmd := fmt.Sprintf(
		`notify-send -u %s -a "Screentime Guardian" -i dialog-warning %s %s`,
		urgency,
		strconv.Quote(summary),
		strconv.Quote(body),
	)

	cmd := exec.Command("runuser", "-u", username, "--", "sh", "-c", notifyCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to send notification to %s: %w (output: %s)", username, err, output)
	}

	return nil
}

// LogNotifier logs notifications (useful for testing/debugging)
type LogNotifier struct{}

// NewLogNotifier creates a log-based notifier
func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// SendWarning logs a warning
func (l *LogNotifier) SendWarning(ctx context.Context, username string, minutesLeft int) error {
	log.Printf("[NOTIFY] User %s: %d minutes remaining", username, minutesLeft)
	return nil
}

// SendLockNotice logs a lock notice
func (l *LogNotifier) SendLockNotice(ctx context.Context, username string) error {
	log.Printf("[NOTIFY] User %s: Session will be locked", username)
	return nil
}

// SendTimeExtended logs a time extension
func (l *LogNotifier) SendTimeExtended(ctx context.Context, username string, minutes int) error {
	log.Printf("[NOTIFY] User %s: Time extended by %d minutes", username, minutes)
	return nil
}
