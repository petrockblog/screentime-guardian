package notifier

import (
	"context"
)

// MockNotifier is a test implementation of Notifier
type MockNotifier struct {
	WarningCalls    []WarningCall
	LockCalls       []string
	ExtensionCalls  []ExtensionCall
	ShouldFailAfter int
	callCount       int
}

type WarningCall struct {
	Username    string
	MinutesLeft int
}

type ExtensionCall struct {
	Username string
	Minutes  int
}

func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		WarningCalls:   make([]WarningCall, 0),
		LockCalls:      make([]string, 0),
		ExtensionCalls: make([]ExtensionCall, 0),
	}
}

func (m *MockNotifier) SendWarning(ctx context.Context, username string, minutesLeft int) error {
	m.callCount++
	if m.ShouldFailAfter > 0 && m.callCount > m.ShouldFailAfter {
		return &MockError{Message: "mock warning error"}
	}
	m.WarningCalls = append(m.WarningCalls, WarningCall{username, minutesLeft})
	return nil
}

func (m *MockNotifier) SendLockNotice(ctx context.Context, username string) error {
	m.callCount++
	if m.ShouldFailAfter > 0 && m.callCount > m.ShouldFailAfter {
		return &MockError{Message: "mock lock error"}
	}
	m.LockCalls = append(m.LockCalls, username)
	return nil
}

func (m *MockNotifier) SendTimeExtended(ctx context.Context, username string, minutes int) error {
	m.callCount++
	if m.ShouldFailAfter > 0 && m.callCount > m.ShouldFailAfter {
		return &MockError{Message: "mock extension error"}
	}
	m.ExtensionCalls = append(m.ExtensionCalls, ExtensionCall{username, minutes})
	return nil
}

type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}
