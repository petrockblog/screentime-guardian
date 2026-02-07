package api

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petrockblog/screentime-guardian/internal/config"
	"github.com/petrockblog/screentime-guardian/internal/notifier"
	"github.com/petrockblog/screentime-guardian/internal/storage"
)

// TestTemplatesParsing tests that templates can be parsed with required functions
func TestTemplatesParsing(t *testing.T) {
	// Templates need divf function for division
	funcMap := template.FuncMap{
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}

	tmpl := template.New("").Funcs(funcMap)
	_, err := tmpl.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}
}

// TestRouterBasics tests the router without D-Bus dependencies
func TestRouterBasics(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	cfg := config.Default()
	cfg.AdminPassword = "" // No auth for testing

	mockNotifier := notifier.NewChain()

	// Create router with nil logind (won't be used in these tests)
	router := NewRouter(store, nil, mockNotifier, cfg)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "Dashboard",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
		{
			name:       "Users page",
			path:       "/users",
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
		{
			name:       "History page",
			path:       "/history",
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
		{
			name:       "Settings page",
			path:       "/settings",
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("Expected body to contain %q", tt.wantBody)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	cfg := config.Default()
	cfg.AdminPassword = "secret"

	mockNotifier := notifier.NewChain()
	router := NewRouter(store, nil, mockNotifier, cfg)

	// Without auth
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without auth, got %d", w.Code)
	}

	// With correct auth
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with correct auth, got %d", w.Code)
	}

	// With wrong password
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "wrong")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 with wrong password, got %d", w.Code)
	}
}

func TestStaticFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.New(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	cfg := config.Default()
	mockNotifier := notifier.NewChain()

	router := NewRouter(store, nil, mockNotifier, cfg)

	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should either find the file (200) or not (404), but not crash
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Static file handler returned unexpected status: %d", w.Code)
	}
}
