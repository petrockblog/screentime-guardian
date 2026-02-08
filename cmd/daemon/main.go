package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/petrockblog/screentime-guardian/internal/api"
	"github.com/petrockblog/screentime-guardian/internal/config"
	"github.com/petrockblog/screentime-guardian/internal/dbus"
	"github.com/petrockblog/screentime-guardian/internal/mdns"
	"github.com/petrockblog/screentime-guardian/internal/notifier"
	"github.com/petrockblog/screentime-guardian/internal/scheduler"
	"github.com/petrockblog/screentime-guardian/internal/storage"
)

var Version = "dev"

func main() {
	configPath := flag.String("config", "/etc/parental-control/config.yaml", "Path to config file")
	flag.Parse()

	log.Printf("Screentime Guardian Daemon %s starting...", Version)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: Could not load config from %s: %v (using defaults)", *configPath, err)
		cfg = config.Default()
	}

	// Initialize storage
	store, err := storage.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize D-Bus connections
	logindClient, err := dbus.NewLogindClient()
	if err != nil {
		log.Fatalf("Failed to connect to logind: %v", err)
	}
	defer logindClient.Close()

	dbusNotifier, err := dbus.NewNotifier()
	if err != nil {
		log.Printf("Warning: Desktop notifications unavailable: %v", err)
		dbusNotifier = nil
	}

	// Create notifier chain (allows adding Telegram later)
	var notifiers []notifier.Notifier
	if dbusNotifier != nil {
		notifiers = append(notifiers, notifier.NewDBusNotifier(dbusNotifier))
	}
	notifierChain := notifier.NewChain(notifiers...)

	// Initialize scheduler
	sched := scheduler.New(store, logindClient, notifierChain, cfg)
	go sched.Run(context.Background())

	// Extract port from listen address for mDNS
	port := 8080
	if parts := strings.Split(cfg.ListenAddr, ":"); len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	// Start mDNS advertisement
	mdnsService, err := mdns.Start(context.Background(), cfg.MDNSHostname, strconv.Itoa(port))
	if err != nil {
		log.Printf("Warning: mDNS advertisement failed: %v", err)
	}

	// Initialize web API
	router := api.NewRouter(store, logindClient, notifierChain, cfg)

	// Start HTTP server
	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Web interface available at http://%s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sched.Stop()
	if mdnsService != nil {
		mdnsService.Stop()
	}
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
}
