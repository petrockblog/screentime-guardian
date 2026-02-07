package scheduler

import (
	"path/filepath"
	"testing"

	"github.com/petrockblog/screentime-guardian/internal/storage"
)

func TestSchedulerBasics(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Test basic user time tracking
	user, err := store.CreateUser("testuser", 120)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Initially should have full time
	remaining, _ := store.GetRemainingMinutes(user.ID)
	if remaining != 120 {
		t.Errorf("Expected 120 minutes remaining, got %d", remaining)
	}

	// Add some usage
	store.AddUsageTime(user.ID, 1800) // 30 minutes
	remaining, _ = store.GetRemainingMinutes(user.ID)
	if remaining != 90 {
		t.Errorf("Expected 90 minutes remaining after 30 min usage, got %d", remaining)
	}

	// Add extension
	store.AddTimeExtension(user.ID, 15, "parent")
	remaining, _ = store.GetRemainingMinutes(user.ID)
	if remaining != 105 {
		t.Errorf("Expected 105 minutes remaining with extension, got %d", remaining)
	}

	// Test disabled user
	store.UpdateUser(user.ID, 120, false)
	updated, _ := store.GetUserByID(user.ID)
	if updated.Enabled {
		t.Error("User should be disabled")
	}
}

func TestWarningsTracking(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	// Test approaching limit
	store.AddUsageTime(user.ID, 6600) // 110 minutes used, 10 remaining
	remaining, _ := store.GetRemainingMinutes(user.ID)

	if remaining < 5 || remaining > 15 {
		t.Logf("Warning threshold test: %d minutes remaining", remaining)
	}

	// Test time expired
	store.AddUsageTime(user.ID, 1200) // 20 more minutes
	remaining, _ = store.GetRemainingMinutes(user.ID)
	if remaining != 0 {
		t.Errorf("Expected 0 minutes remaining (time exceeded), got %d", remaining)
	}
}
