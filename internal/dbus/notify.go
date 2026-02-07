package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// Notifier provides desktop notification capabilities via D-Bus
type Notifier struct {
	conn *dbus.Conn
}

// NewNotifier creates a new desktop notification client
func NewNotifier() (*Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	return &Notifier{conn: conn}, nil
}

// Close closes the D-Bus connection
func (n *Notifier) Close() error {
	if n.conn != nil {
		return n.conn.Close()
	}
	return nil
}

// Urgency levels for notifications
type Urgency byte

const (
	UrgencyLow      Urgency = 0
	UrgencyNormal   Urgency = 1
	UrgencyCritical Urgency = 2
)

// Notify sends a desktop notification
func (n *Notifier) Notify(summary, body string, urgency Urgency) (uint32, error) {
	obj := n.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(byte(urgency)),
	}

	var notificationID uint32
	err := obj.Call(
		"org.freedesktop.Notifications.Notify",
		0,
		"Screentime Guardian",
		uint32(0),
		"dialog-warning",
		summary,
		body,
		[]string{},
		hints,
		int32(-1),
	).Store(&notificationID)

	if err != nil {
		return 0, fmt.Errorf("failed to send notification: %w", err)
	}

	return notificationID, nil
}

// NotifyWarning sends a warning notification about remaining time
func (n *Notifier) NotifyWarning(minutesLeft int) (uint32, error) {
	summary := "Time Warning"
	body := fmt.Sprintf("You have %d minute(s) of screen time remaining.", minutesLeft)

	urgency := UrgencyNormal
	if minutesLeft <= 1 {
		urgency = UrgencyCritical
	}

	return n.Notify(summary, body, urgency)
}

// NotifyLock sends a notification that the session will be locked
func (n *Notifier) NotifyLock() (uint32, error) {
	return n.Notify(
		"Time's Up!",
		"Your screen time has ended. The session will now be locked.",
		UrgencyCritical,
	)
}

// NotifyExtended sends a notification that time has been extended
func (n *Notifier) NotifyExtended(minutes int) (uint32, error) {
	return n.Notify(
		"Time Extended",
		fmt.Sprintf("Your screen time has been extended by %d minutes.", minutes),
		UrgencyNormal,
	)
}
