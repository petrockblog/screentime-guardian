package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// ListenAddr is the address for the web interface
	ListenAddr string `yaml:"listen_addr"`

	// DatabasePath is the path to the SQLite database
	DatabasePath string `yaml:"database_path"`

	// AdminPassword is the password for the web interface
	AdminPassword string `yaml:"admin_password"`

	// MDNSHostname is the hostname advertised via mDNS (e.g., "screentime-guardian")
	// If empty, defaults to "screentime-guardian-{system-hostname}"
	// This allows multiple instances in the same network
	MDNSHostname string `yaml:"mdns_hostname"`

	// WarningIntervals defines when to warn before lockout (in minutes)
	WarningIntervals []int `yaml:"warning_intervals"`

	// CheckInterval is how often to check time limits
	CheckInterval time.Duration `yaml:"check_interval"`

	// GracePeriod is extra time after limit before hard lock
	GracePeriod time.Duration `yaml:"grace_period"`
}

// Default returns a configuration with sensible defaults
func Default() *Config {
	return &Config{
		ListenAddr:       ":8080",
		DatabasePath:     "/var/lib/parental-control/data.db",
		AdminPassword:    "",          // Must be set on first run
		WarningIntervals: []int{5, 1}, // Warn at 5 minutes and 1 minute
		CheckInterval:    30 * time.Second,
		GracePeriod:      1 * time.Minute,
	}
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to a YAML file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
