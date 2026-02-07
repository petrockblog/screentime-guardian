package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/petrockblog/screentime-guardian/internal/config"
	"github.com/petrockblog/screentime-guardian/internal/dbus"
	"github.com/petrockblog/screentime-guardian/internal/notifier"
	"github.com/petrockblog/screentime-guardian/internal/storage"
)

// Scheduler manages time tracking and enforcement for all users
type Scheduler struct {
	store    *storage.Storage
	logind   *dbus.LogindClient
	notifier *notifier.Chain
	config   *config.Config

	warningsSent   map[string]map[int]bool
	mu             sync.Mutex
	activeSessions map[string]time.Time
	lastCheck      time.Time

	stop chan struct{}
	done chan struct{}
}

// New creates a new scheduler
func New(store *storage.Storage, logind *dbus.LogindClient, notifier *notifier.Chain, cfg *config.Config) *Scheduler {
	return &Scheduler{
		store:          store,
		logind:         logind,
		notifier:       notifier,
		config:         cfg,
		warningsSent:   make(map[string]map[int]bool),
		activeSessions: make(map[string]time.Time),
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

// Run starts the scheduler loop
func (s *Scheduler) Run(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	log.Printf("Scheduler started with check interval %v", s.config.CheckInterval)

	s.check(ctx)

	for {
		select {
		case <-ticker.C:
			s.check(ctx)
		case <-s.stop:
			log.Println("Scheduler stopping...")
			return
		case <-ctx.Done():
			log.Println("Scheduler context cancelled...")
			return
		}
	}
}

// Stop signals the scheduler to stop
func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Scheduler) check(ctx context.Context) {
	now := time.Now()
	elapsed := now.Sub(s.lastCheck)
	s.lastCheck = now

	if now.Hour() == 0 && now.Minute() < 1 {
		s.mu.Lock()
		s.warningsSent = make(map[string]map[int]bool)
		s.mu.Unlock()
	}

	users, err := s.store.ListUsers()
	if err != nil {
		log.Printf("Failed to list users: %v", err)
		return
	}

	sessions, err := s.logind.ListSessions()
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
		return
	}

	loggedIn := make(map[string]bool)
	for _, session := range sessions {
		loggedIn[session.UserName] = true
	}

	for _, user := range users {
		if !user.Enabled {
			continue
		}

		isLoggedIn := loggedIn[user.Username]

		if isLoggedIn && elapsed > 0 {
			s.mu.Lock()
			if _, ok := s.activeSessions[user.Username]; !ok {
				s.activeSessions[user.Username] = now
				log.Printf("User %s session started", user.Username)
			}
			s.mu.Unlock()

			seconds := int(elapsed.Seconds())
			if err := s.store.AddUsageTime(user.ID, seconds); err != nil {
				log.Printf("Failed to add usage time for %s: %v", user.Username, err)
			}
		} else if !isLoggedIn {
			s.mu.Lock()
			if _, ok := s.activeSessions[user.Username]; ok {
				delete(s.activeSessions, user.Username)
				log.Printf("User %s session ended", user.Username)
			}
			s.mu.Unlock()
		}

		if !isLoggedIn {
			continue
		}

		remaining, err := s.store.GetRemainingMinutes(user.ID)
		if err != nil {
			log.Printf("Failed to get remaining time for %s: %v", user.Username, err)
			continue
		}

		if remaining <= 0 {
			s.handleTimeExpired(ctx, user.Username)
			continue
		}

		s.checkWarnings(ctx, user.Username, remaining)
	}
}

func (s *Scheduler) handleTimeExpired(ctx context.Context, username string) {
	log.Printf("Time expired for user %s, locking session", username)

	if err := s.notifier.SendLockNotice(ctx, username); err != nil {
		log.Printf("Failed to send lock notice to %s: %v", username, err)
	}

	time.Sleep(3 * time.Second)

	if err := s.logind.LockUserSessions(username); err != nil {
		log.Printf("Failed to lock sessions for %s: %v", username, err)
	}
}

func (s *Scheduler) checkWarnings(ctx context.Context, username string, remaining int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.warningsSent[username] == nil {
		s.warningsSent[username] = make(map[int]bool)
	}

	for _, interval := range s.config.WarningIntervals {
		if remaining <= interval && !s.warningsSent[username][interval] {
			log.Printf("Sending %d minute warning to %s", interval, username)
			if err := s.notifier.SendWarning(ctx, username, remaining); err != nil {
				log.Printf("Failed to send warning to %s: %v", username, err)
			}
			s.warningsSent[username][interval] = true
		}
	}
}

// ResetWarnings clears warning state for a user
func (s *Scheduler) ResetWarnings(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.warningsSent, username)
}

// ForceCheck triggers an immediate check cycle
func (s *Scheduler) ForceCheck(ctx context.Context) {
	s.check(ctx)
}

// UserStatus represents current status for a user
type UserStatus struct {
	Username       string
	IsLoggedIn     bool
	RemainingMins  int
	UsedMins       int
	DailyLimitMins int
	ExtensionMins  int
	Enabled        bool
}

// GetAllStatus returns status for all tracked users
func (s *Scheduler) GetAllStatus(ctx context.Context) ([]UserStatus, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return nil, err
	}

	sessions, err := s.logind.ListSessions()
	if err != nil {
		return nil, err
	}

	loggedIn := make(map[string]bool)
	for _, session := range sessions {
		loggedIn[session.UserName] = true
	}

	var statuses []UserStatus
	for _, user := range users {
		remaining, _ := s.store.GetRemainingMinutes(user.ID)
		usedSecs, _ := s.store.GetTodayUsageSeconds(user.ID)
		extensions, _ := s.store.GetTodayExtensions(user.ID)

		statuses = append(statuses, UserStatus{
			Username:       user.Username,
			IsLoggedIn:     loggedIn[user.Username],
			RemainingMins:  remaining,
			UsedMins:       usedSecs / 60,
			DailyLimitMins: user.DailyLimitMins,
			ExtensionMins:  extensions,
			Enabled:        user.Enabled,
		})
	}

	return statuses, nil
}
