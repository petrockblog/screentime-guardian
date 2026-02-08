package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

// SessionStore manages authentication sessions
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// Session represents an authenticated session
type Session struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewSessionStore creates a new session store
func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*Session),
	}

	// Start cleanup goroutine
	go store.cleanupExpired()

	return store
}

// Create creates a new session and returns a token
func (s *SessionStore) Create() (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	session := &Session{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()

	return token, nil
}

// Validate checks if a token is valid
func (s *SessionStore) Validate(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]
	if !exists {
		return false
	}

	return time.Now().Before(session.ExpiresAt)
}

// Delete removes a session
func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// cleanupExpired removes expired sessions every hour
func (s *SessionStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}

// generateToken creates a cryptographically secure random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SessionAuthMiddleware checks for valid session cookie or falls back to basic auth
func SessionAuthMiddleware(sessionStore *SessionStore, adminPassword string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if no password is set
			if adminPassword == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for session cookie
			cookie, err := r.Cookie("session_token")
			if err == nil && sessionStore.Validate(cookie.Value) {
				// Valid session, allow access
				next.ServeHTTP(w, r)
				return
			}

			// No valid session, check basic auth
			user, pass, ok := r.BasicAuth()
			if !ok || user != "admin" || pass != adminPassword {
				w.Header().Set("WWW-Authenticate", `Basic realm="Screentime Guardian"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Basic auth succeeded, create session
			token, err := sessionStore.Create()
			if err == nil {
				// Set session cookie (7 days, httpOnly, secure if HTTPS)
				http.SetCookie(w, &http.Cookie{
					Name:     "session_token",
					Value:    token,
					Path:     "/",
					MaxAge:   7 * 24 * 60 * 60, // 7 days in seconds
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
					// Secure: true, // Uncomment if using HTTPS
				})
			}

			next.ServeHTTP(w, r)
		})
	}
}
