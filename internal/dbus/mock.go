package dbus

// MockLogindClient is a test implementation of LogindClient
type MockLogindClient struct {
	Sessions           []Session
	LockedSessions     []string
	TerminatedSessions []string
	ShouldError        bool
}

// NewMockLogindClient creates a new mock logind client
func NewMockLogindClient() *MockLogindClient {
	return &MockLogindClient{
		Sessions:           make([]Session, 0),
		LockedSessions:     make([]string, 0),
		TerminatedSessions: make([]string, 0),
	}
}

// Close does nothing in the mock
func (m *MockLogindClient) Close() error {
	return nil
}

// ListSessions returns the mock sessions
func (m *MockLogindClient) ListSessions() ([]Session, error) {
	if m.ShouldError {
		return nil, &MockError{Message: "mock list sessions error"}
	}
	return m.Sessions, nil
}

// LockSession adds the session ID to the locked list
func (m *MockLogindClient) LockSession(sessionID string) error {
	if m.ShouldError {
		return &MockError{Message: "mock lock error"}
	}
	m.LockedSessions = append(m.LockedSessions, sessionID)
	return nil
}

// LockSessions locks all sessions
func (m *MockLogindClient) LockSessions() error {
	if m.ShouldError {
		return &MockError{Message: "mock lock all error"}
	}
	for _, session := range m.Sessions {
		m.LockedSessions = append(m.LockedSessions, session.ID)
	}
	return nil
}

// TerminateSession adds the session ID to the terminated list
func (m *MockLogindClient) TerminateSession(sessionID string) error {
	if m.ShouldError {
		return &MockError{Message: "mock terminate error"}
	}
	m.TerminatedSessions = append(m.TerminatedSessions, sessionID)
	return nil
}

// LockUserSessions locks all sessions for a specific user
func (m *MockLogindClient) LockUserSessions(username string) error {
	sessions, err := m.ListSessions()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.UserName == username {
			if err := m.LockSession(session.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// TerminateUserSessions terminates all sessions for a specific user
func (m *MockLogindClient) TerminateUserSessions(username string) error {
	sessions, err := m.ListSessions()
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.UserName == username {
			if err := m.TerminateSession(session.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsUserLoggedIn checks if a user has any active sessions
func (m *MockLogindClient) IsUserLoggedIn(username string) (bool, error) {
	sessions, err := m.ListSessions()
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
func (m *MockLogindClient) GetUserSessions(username string) ([]Session, error) {
	sessions, err := m.ListSessions()
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

// MockNotifier is a test implementation of desktop notifications
type MockNotifier struct {
	Notifications []MockNotification
	ShouldError   bool
}

type MockNotification struct {
	Summary string
	Body    string
	Urgency Urgency
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		Notifications: make([]MockNotification, 0),
	}
}

// Close does nothing in the mock
func (m *MockNotifier) Close() error {
	return nil
}

// Notify records the notification
func (m *MockNotifier) Notify(summary, body string, urgency Urgency) (uint32, error) {
	if m.ShouldError {
		return 0, &MockError{Message: "mock notify error"}
	}
	m.Notifications = append(m.Notifications, MockNotification{summary, body, urgency})
	return uint32(len(m.Notifications)), nil
}

// NotifyWarning records a warning notification
func (m *MockNotifier) NotifyWarning(minutesLeft int) (uint32, error) {
	summary := "Time Warning"
	body := ""
	urgency := UrgencyNormal
	if minutesLeft <= 1 {
		urgency = UrgencyCritical
	}
	return m.Notify(summary, body, urgency)
}

// NotifyLock records a lock notification
func (m *MockNotifier) NotifyLock() (uint32, error) {
	return m.Notify("Time's Up!", "Your screen time has ended.", UrgencyCritical)
}

// NotifyExtended records an extension notification
func (m *MockNotifier) NotifyExtended(minutes int) (uint32, error) {
	return m.Notify("Time Extended", "", UrgencyNormal)
}

type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}
