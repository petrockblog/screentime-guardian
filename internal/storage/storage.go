package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Storage handles all database operations
type Storage struct {
	db *sql.DB
}

// User represents a child user account
type User struct {
	ID             int64
	Username       string
	DailyLimitMins int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UsageRecord represents daily usage for a user
type UsageRecord struct {
	ID          int64
	UserID      int64
	Date        string
	UsedSeconds int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TimeExtension represents a time extension granted to a user
type TimeExtension struct {
	ID        int64
	UserID    int64
	Date      string
	Minutes   int
	GrantedBy string
	CreatedAt time.Time
}

// New creates a new storage instance and initializes the database
func New(dbPath string) (*Storage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) migrate() error {
	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			daily_limit_mins INTEGER NOT NULL DEFAULT 120,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS usage_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			used_seconds INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(user_id, date)
		);

		CREATE TABLE IF NOT EXISTS time_extensions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			minutes INTEGER NOT NULL,
			granted_by TEXT NOT NULL DEFAULT 'parent',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_usage_log_user_date ON usage_log(user_id, date);
		CREATE INDEX IF NOT EXISTS idx_extensions_user_date ON time_extensions(user_id, date);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateUser creates a new user
func (s *Storage) CreateUser(username string, dailyLimitMins int) (*User, error) {
	result, err := s.db.Exec(
		`INSERT INTO users (username, daily_limit_mins) VALUES (?, ?)`,
		username, dailyLimitMins,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, _ := result.LastInsertId()
	return s.GetUserByID(id)
}

// GetUserByID retrieves a user by ID
func (s *Storage) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, daily_limit_mins, enabled, created_at, updated_at 
		 FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.DailyLimitMins, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *Storage) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, daily_limit_mins, enabled, created_at, updated_at 
		 FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.DailyLimitMins, &user.Enabled, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ListUsers returns all users
func (s *Storage) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(
		`SELECT id, username, daily_limit_mins, enabled, created_at, updated_at 
		 FROM users ORDER BY username`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.DailyLimitMins, &user.Enabled, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateUser updates a user's settings
func (s *Storage) UpdateUser(id int64, dailyLimitMins int, enabled bool) error {
	_, err := s.db.Exec(
		`UPDATE users SET daily_limit_mins = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP 
		 WHERE id = ?`,
		dailyLimitMins, enabled, id,
	)
	return err
}

// DeleteUser deletes a user
func (s *Storage) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// AddUsageTime adds seconds to today's usage for a user
func (s *Storage) AddUsageTime(userID int64, seconds int) error {
	today := time.Now().Format("2006-01-02")

	_, err := s.db.Exec(
		`INSERT INTO usage_log (user_id, date, used_seconds) 
		 VALUES (?, ?, ?)
		 ON CONFLICT(user_id, date) DO UPDATE SET 
		 used_seconds = used_seconds + ?,
		 updated_at = CURRENT_TIMESTAMP`,
		userID, today, seconds, seconds,
	)
	return err
}

// GetTodayUsageSeconds returns the number of seconds used today
func (s *Storage) GetTodayUsageSeconds(userID int64) (int, error) {
	today := time.Now().Format("2006-01-02")

	var seconds int
	err := s.db.QueryRow(
		`SELECT COALESCE(used_seconds, 0) FROM usage_log 
		 WHERE user_id = ? AND date = ?`,
		userID, today,
	).Scan(&seconds)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	return seconds, err
}

// GetUsageHistory returns usage records for a user over the past N days
func (s *Storage) GetUsageHistory(userID int64, days int) ([]*UsageRecord, error) {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := s.db.Query(
		`SELECT id, user_id, date, used_seconds, created_at, updated_at 
		 FROM usage_log WHERE user_id = ? AND date >= ? ORDER BY date DESC`,
		userID, startDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}
	defer rows.Close()

	var records []*UsageRecord
	for rows.Next() {
		record := &UsageRecord{}
		if err := rows.Scan(&record.ID, &record.UserID, &record.Date, &record.UsedSeconds, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan usage record: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

// AddTimeExtension adds a time extension for a user
func (s *Storage) AddTimeExtension(userID int64, minutes int, grantedBy string) error {
	today := time.Now().Format("2006-01-02")

	_, err := s.db.Exec(
		`INSERT INTO time_extensions (user_id, date, minutes, granted_by) 
		 VALUES (?, ?, ?, ?)`,
		userID, today, minutes, grantedBy,
	)
	return err
}

// GetTodayExtensions returns total extension minutes for today
func (s *Storage) GetTodayExtensions(userID int64) (int, error) {
	today := time.Now().Format("2006-01-02")

	var minutes int
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(minutes), 0) FROM time_extensions 
		 WHERE user_id = ? AND date = ?`,
		userID, today,
	).Scan(&minutes)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	return minutes, err
}

// GetSetting retrieves a setting value
func (s *Storage) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting sets a setting value
func (s *Storage) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?`,
		key, value, value,
	)
	return err
}

// GetRemainingMinutes calculates remaining minutes for a user today
func (s *Storage) GetRemainingMinutes(userID int64) (int, error) {
	user, err := s.GetUserByID(userID)
	if err != nil || user == nil {
		return 0, err
	}

	usedSeconds, err := s.GetTodayUsageSeconds(userID)
	if err != nil {
		return 0, err
	}

	extensions, err := s.GetTodayExtensions(userID)
	if err != nil {
		return 0, err
	}

	totalLimitMins := user.DailyLimitMins + extensions
	usedMins := usedSeconds / 60
	remaining := totalLimitMins - usedMins

	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}
