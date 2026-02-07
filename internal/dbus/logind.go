package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// LogindClient provides access to systemd-logind session management
type LogindClient struct {
	conn *dbus.Conn
}

// Session represents a user session from logind
type Session struct {
	ID       string
	UserID   uint32
	UserName string
	Seat     string
	Path     dbus.ObjectPath
}

// NewLogindClient creates a new connection to systemd-logind
func NewLogindClient() (*LogindClient, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}

	return &LogindClient{conn: conn}, nil
}

// Close closes the D-Bus connection
func (c *LogindClient) Close() error {
	return c.conn.Close()
}

// ListSessions returns all active user sessions
func (c *LogindClient) ListSessions() ([]Session, error) {
	obj := c.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")

	var result [][]interface{}
	err := obj.Call("org.freedesktop.login1.Manager.ListSessions", 0).Store(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]Session, 0, len(result))
	for _, s := range result {
		if len(s) < 5 {
			continue
		}

		sessions = append(sessions, Session{
			ID:       s[0].(string),
			UserID:   s[1].(uint32),
			UserName: s[2].(string),
			Seat:     s[3].(string),
			Path:     s[4].(dbus.ObjectPath),
		})
	}

	return sessions, nil
}

// LockSession locks a specific session by ID
func (c *LogindClient) LockSession(sessionID string) error {
	obj := c.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")

	call := obj.Call("org.freedesktop.login1.Manager.LockSession", 0, sessionID)
	if call.Err != nil {
		return fmt.Errorf("failed to lock session %s: %w", sessionID, call.Err)
	}

	return nil
}

// LockSessions locks all sessions
func (c *LogindClient) LockSessions() error {
	obj := c.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")

	call := obj.Call("org.freedesktop.login1.Manager.LockSessions", 0)
	if call.Err != nil {
		return fmt.Errorf("failed to lock all sessions: %w", call.Err)
	}

	return nil
}

// TerminateSession forcefully terminates a session (logs user out)
func (c *LogindClient) TerminateSession(sessionID string) error {
	obj := c.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")

	call := obj.Call("org.freedesktop.login1.Manager.TerminateSession", 0, sessionID)
	if call.Err != nil {
		return fmt.Errorf("failed to terminate session %s: %w", sessionID, call.Err)
	}

	return nil
}

// LockUserSessions locks all sessions for a specific user
func (c *LogindClient) LockUserSessions(username string) error {
	sessions, err := c.ListSessions()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.UserName == username {
			if err := c.LockSession(session.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// TerminateUserSessions terminates all sessions for a specific user
func (c *LogindClient) TerminateUserSessions(username string) error {
	sessions, err := c.ListSessions()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.UserName == username {
			if err := c.TerminateSession(session.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsUserLoggedIn checks if a user has any active sessions
func (c *LogindClient) IsUserLoggedIn(username string) (bool, error) {
	sessions, err := c.ListSessions()
	if err != nil {
		return false, err
	}

	for _, session := range sessions {
		if session.UserName == username {
			return true, nil
		}
	}

	return false, nil
}

// GetUserSessions returns all sessions for a specific user
func (c *LogindClient) GetUserSessions(username string) ([]Session, error) {
	sessions, err := c.ListSessions()
	if err != nil {
		return nil, err
	}

	var userSessions []Session
	for _, session := range sessions {
		if session.UserName == username {
			userSessions = append(userSessions, session)
		}
	}

	return userSessions, nil
}
