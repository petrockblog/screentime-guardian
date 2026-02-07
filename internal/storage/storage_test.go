package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestCreateAndGetUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, err := store.CreateUser("testuser", 120)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", user.Username)
	}

	if user.DailyLimitMins != 120 {
		t.Errorf("Expected daily limit 120, got %d", user.DailyLimitMins)
	}

	if !user.Enabled {
		t.Error("Expected user to be enabled by default")
	}

	retrieved, err := store.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if retrieved.Username != user.Username {
		t.Errorf("Retrieved user mismatch: expected %s, got %s", user.Username, retrieved.Username)
	}

	byUsername, err := store.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if byUsername.ID != user.ID {
		t.Errorf("Retrieved user ID mismatch: expected %d, got %d", user.ID, byUsername.ID)
	}
}

func TestListUsers(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}

	store.CreateUser("user1", 60)
	store.CreateUser("user2", 90)
	store.CreateUser("user3", 120)

	users, err = store.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestUpdateUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	err = store.UpdateUser(user.ID, 180, false)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	updated, _ := store.GetUserByID(user.ID)
	if updated.DailyLimitMins != 180 {
		t.Errorf("Expected daily limit 180, got %d", updated.DailyLimitMins)
	}

	if updated.Enabled {
		t.Error("Expected user to be disabled")
	}
}

func TestDeleteUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	err = store.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	retrieved, _ := store.GetUserByID(user.ID)
	if retrieved != nil {
		t.Error("Expected user to be deleted, but still exists")
	}
}

func TestUsageTracking(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	err = store.AddUsageTime(user.ID, 1800)
	if err != nil {
		t.Fatalf("Failed to add usage time: %v", err)
	}

	usedSeconds, err := store.GetTodayUsageSeconds(user.ID)
	if err != nil {
		t.Fatalf("Failed to get usage: %v", err)
	}

	if usedSeconds != 1800 {
		t.Errorf("Expected 1800 seconds used, got %d", usedSeconds)
	}

	store.AddUsageTime(user.ID, 600)

	usedSeconds, _ = store.GetTodayUsageSeconds(user.ID)
	if usedSeconds != 2400 {
		t.Errorf("Expected 2400 seconds used, got %d", usedSeconds)
	}
}

func TestTimeExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	err = store.AddTimeExtension(user.ID, 30, "parent")
	if err != nil {
		t.Fatalf("Failed to add time extension: %v", err)
	}

	extensions, err := store.GetTodayExtensions(user.ID)
	if err != nil {
		t.Fatalf("Failed to get extensions: %v", err)
	}

	if extensions != 30 {
		t.Errorf("Expected 30 minutes extension, got %d", extensions)
	}

	store.AddTimeExtension(user.ID, 15, "schedule")

	extensions, _ = store.GetTodayExtensions(user.ID)
	if extensions != 45 {
		t.Errorf("Expected 45 minutes total extension, got %d", extensions)
	}
}

func TestGetRemainingMinutes(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	remaining, err := store.GetRemainingMinutes(user.ID)
	if err != nil {
		t.Fatalf("Failed to get remaining minutes: %v", err)
	}

	if remaining != 120 {
		t.Errorf("Expected 120 minutes remaining, got %d", remaining)
	}

	store.AddUsageTime(user.ID, 1800)

	remaining, _ = store.GetRemainingMinutes(user.ID)
	if remaining != 90 {
		t.Errorf("Expected 90 minutes remaining, got %d", remaining)
	}

	store.AddTimeExtension(user.ID, 15, "parent")

	remaining, _ = store.GetRemainingMinutes(user.ID)
	if remaining != 105 {
		t.Errorf("Expected 105 minutes remaining (90 + 15 extension), got %d", remaining)
	}

	store.AddUsageTime(user.ID, 7500)

	remaining, _ = store.GetRemainingMinutes(user.ID)
	if remaining != 0 {
		t.Errorf("Expected 0 minutes remaining (over limit), got %d", remaining)
	}
}

func TestUsageHistory(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	user, _ := store.CreateUser("testuser", 120)

	store.AddUsageTime(user.ID, 3600)

	history, err := store.GetUsageHistory(user.ID, 7)
	if err != nil {
		t.Fatalf("Failed to get usage history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}

	if history[0].UsedSeconds != 3600 {
		t.Errorf("Expected 3600 seconds in history, got %d", history[0].UsedSeconds)
	}

	today := time.Now().Format("2006-01-02")
	if history[0].Date != today {
		t.Errorf("Expected date %s, got %s", today, history[0].Date)
	}
}

func TestSettings(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	err = store.SetSetting("test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	value, err := store.GetSetting("test_key")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected test_value, got %s", value)
	}

	store.SetSetting("test_key", "updated_value")

	value, _ = store.GetSetting("test_key")
	if value != "updated_value" {
		t.Errorf("Expected updated_value, got %s", value)
	}

	value, _ = store.GetSetting("nonexistent")
	if value != "" {
		t.Errorf("Expected empty string for nonexistent setting, got %s", value)
	}
}

func TestDuplicateUsername(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	_, err = store.CreateUser("testuser", 120)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	_, err = store.CreateUser("testuser", 90)
	if err == nil {
		t.Error("Expected error when creating duplicate username, got nil")
	}
}
