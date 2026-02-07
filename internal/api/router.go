package api

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/petrockblog/screentime-guardian/internal/config"
	"github.com/petrockblog/screentime-guardian/internal/dbus"
	"github.com/petrockblog/screentime-guardian/internal/notifier"
	"github.com/petrockblog/screentime-guardian/internal/storage"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server holds the API dependencies
type Server struct {
	store    *storage.Storage
	logind   *dbus.LogindClient
	notifier *notifier.Chain
	config   *config.Config
	tmpl     *template.Template
}

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(store *storage.Storage, logind *dbus.LogindClient, notifier *notifier.Chain, cfg *config.Config) http.Handler {
	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"divf": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}

	tmpl := template.New("").Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	s := &Server{
		store:    store,
		logind:   logind,
		notifier: notifier,
		config:   cfg,
		tmpl:     tmpl,
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Basic auth if password is set
	if cfg.AdminPassword != "" {
		r.Use(middleware.BasicAuth("Screentime Guardian", map[string]string{
			"admin": cfg.AdminPassword,
		}))
	}

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Pages
	r.Get("/", s.handleDashboard)
	r.Get("/users", s.handleUsers)
	r.Get("/users/{id}", s.handleUserDetail)
	r.Get("/history", s.handleHistory)
	r.Get("/settings", s.handleSettings)

	// HTMX partials
	r.Get("/partials/status", s.handleStatusPartial)
	r.Get("/partials/users", s.handleUsersPartial)

	// API endpoints
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", s.apiGetStatus)
		r.Post("/users", s.apiCreateUser)
		r.Put("/users/{id}", s.apiUpdateUser)
		r.Delete("/users/{id}", s.apiDeleteUser)
		r.Post("/users/{id}/extend", s.apiExtendTime)
		r.Post("/users/{id}/lock", s.apiLockUser)
		r.Post("/users/{id}/unlock", s.apiUnlockUser)
	})

	return r
}
