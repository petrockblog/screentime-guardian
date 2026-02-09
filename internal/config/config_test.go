package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("Expected ListenAddr :8080, got %s", cfg.ListenAddr)
	}

	if cfg.DatabasePath != "/var/lib/screentime-guardian/data.db" {
		t.Errorf("Expected DatabasePath /var/lib/screentime-guardian/data.db, got %s", cfg.DatabasePath)
	}

	if cfg.AdminPassword != "" {
		t.Errorf("Expected empty AdminPassword, got %s", cfg.AdminPassword)
	}

	if len(cfg.WarningIntervals) != 2 {
		t.Errorf("Expected 2 warning intervals, got %d", len(cfg.WarningIntervals))
	}

	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("Expected CheckInterval 30s, got %v", cfg.CheckInterval)
	}

	if cfg.GracePeriod != 1*time.Minute {
		t.Errorf("Expected GracePeriod 1m, got %v", cfg.GracePeriod)
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	original := &Config{
		ListenAddr:       ":9090",
		DatabasePath:     "/tmp/test.db",
		AdminPassword:    "test-password",
		WarningIntervals: []int{10, 5, 1},
		CheckInterval:    60 * time.Second,
		GracePeriod:      2 * time.Minute,
	}

	if err := original.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.ListenAddr != original.ListenAddr {
		t.Errorf("ListenAddr mismatch: expected %s, got %s", original.ListenAddr, loaded.ListenAddr)
	}

	if loaded.DatabasePath != original.DatabasePath {
		t.Errorf("DatabasePath mismatch: expected %s, got %s", original.DatabasePath, loaded.DatabasePath)
	}

	if loaded.AdminPassword != original.AdminPassword {
		t.Errorf("AdminPassword mismatch: expected %s, got %s", original.AdminPassword, loaded.AdminPassword)
	}

	if loaded.CheckInterval != original.CheckInterval {
		t.Errorf("CheckInterval mismatch: expected %v, got %v", original.CheckInterval, loaded.CheckInterval)
	}

	if loaded.GracePeriod != original.GracePeriod {
		t.Errorf("GracePeriod mismatch: expected %v, got %v", original.GracePeriod, loaded.GracePeriod)
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file, got nil")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: {{{"), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML, got nil")
	}
}
