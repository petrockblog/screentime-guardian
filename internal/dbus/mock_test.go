package dbus

import (
	"testing"
)

func TestMockLogindClient(t *testing.T) {
	client := NewMockLogindClient()

	client.Sessions = []Session{
		{ID: "1", UserID: 1000, UserName: "alice", Seat: "seat0"},
		{ID: "2", UserID: 1001, UserName: "bob", Seat: "seat0"},
		{ID: "3", UserID: 1000, UserName: "alice", Seat: "seat1"},
	}

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	err = client.LockSession("1")
	if err != nil {
		t.Fatalf("LockSession failed: %v", err)
	}

	if len(client.LockedSessions) != 1 || client.LockedSessions[0] != "1" {
		t.Error("Session 1 should be locked")
	}

	err = client.TerminateSession("2")
	if err != nil {
		t.Fatalf("TerminateSession failed: %v", err)
	}

	if len(client.TerminatedSessions) != 1 || client.TerminatedSessions[0] != "2" {
		t.Error("Session 2 should be terminated")
	}

	err = client.LockUserSessions("alice")
	if err != nil {
		t.Fatalf("LockUserSessions failed: %v", err)
	}

	if len(client.LockedSessions) < 3 {
		t.Errorf("Expected at least 3 locked sessions, got %d", len(client.LockedSessions))
	}

	loggedIn, err := client.IsUserLoggedIn("alice")
	if err != nil {
		t.Fatalf("IsUserLoggedIn failed: %v", err)
	}
	if !loggedIn {
		t.Error("alice should be logged in")
	}

	loggedIn, _ = client.IsUserLoggedIn("charlie")
	if loggedIn {
		t.Error("charlie should not be logged in")
	}

	aliceSessions, err := client.GetUserSessions("alice")
	if err != nil {
		t.Fatalf("GetUserSessions failed: %v", err)
	}

	if len(aliceSessions) != 2 {
		t.Errorf("Expected 2 sessions for alice, got %d", len(aliceSessions))
	}
}

func TestMockLogindClientErrors(t *testing.T) {
	client := NewMockLogindClient()
	client.ShouldError = true

	_, err := client.ListSessions()
	if err == nil {
		t.Error("Expected error from ListSessions")
	}

	err = client.LockSession("1")
	if err == nil {
		t.Error("Expected error from LockSession")
	}

	err = client.TerminateSession("1")
	if err == nil {
		t.Error("Expected error from TerminateSession")
	}

	err = client.LockSessions()
	if err == nil {
		t.Error("Expected error from LockSessions")
	}
}

func TestMockNotifier(t *testing.T) {
	notifier := NewMockNotifier()

	id, err := notifier.Notify("Test", "Message", UrgencyNormal)
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	if id != 1 {
		t.Errorf("Expected notification ID 1, got %d", id)
	}

	if len(notifier.Notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifier.Notifications))
	}

	if notifier.Notifications[0].Summary != "Test" {
		t.Errorf("Expected summary 'Test', got '%s'", notifier.Notifications[0].Summary)
	}

	_, err = notifier.NotifyWarning(5)
	if err != nil {
		t.Fatalf("NotifyWarning failed: %v", err)
	}

	if len(notifier.Notifications) != 2 {
		t.Errorf("Expected 2 notifications, got %d", len(notifier.Notifications))
	}

	if notifier.Notifications[1].Urgency != UrgencyNormal {
		t.Error("5 minute warning should have normal urgency")
	}

	_, err = notifier.NotifyWarning(1)
	if err != nil {
		t.Fatalf("NotifyWarning failed: %v", err)
	}

	if notifier.Notifications[2].Urgency != UrgencyCritical {
		t.Error("1 minute warning should have critical urgency")
	}

	_, err = notifier.NotifyLock()
	if err != nil {
		t.Fatalf("NotifyLock failed: %v", err)
	}

	if notifier.Notifications[3].Summary != "Time's Up!" {
		t.Error("Lock notification should have 'Time's Up!' summary")
	}

	_, err = notifier.NotifyExtended(30)
	if err != nil {
		t.Fatalf("NotifyExtended failed: %v", err)
	}

	if notifier.Notifications[4].Summary != "Time Extended" {
		t.Error("Extension notification should have 'Time Extended' summary")
	}
}

func TestMockNotifierErrors(t *testing.T) {
	notifier := NewMockNotifier()
	notifier.ShouldError = true

	_, err := notifier.Notify("Test", "Message", UrgencyNormal)
	if err == nil {
		t.Error("Expected error from Notify")
	}

	_, err = notifier.NotifyWarning(5)
	if err == nil {
		t.Error("Expected error from NotifyWarning")
	}

	_, err = notifier.NotifyLock()
	if err == nil {
		t.Error("Expected error from NotifyLock")
	}

	_, err = notifier.NotifyExtended(30)
	if err == nil {
		t.Error("Expected error from NotifyExtended")
	}
}
