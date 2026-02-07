package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/petrockblog/screentime-guardian/internal/dbus"
	"github.com/petrockblog/screentime-guardian/internal/storage"
)

// --- Page handlers ---

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var sessions []dbus.Session

	// Only query sessions if logind is available
	if s.logind != nil {
		var err error
		sessions, err = s.logind.ListSessions()
		if err != nil {
			sessions = nil // Continue without session info
		}
	}

	loggedIn := make(map[string]bool)
	for _, session := range sessions {
		loggedIn[session.UserName] = true
	}

	type UserData struct {
		ID             int64
		Username       string
		IsLoggedIn     bool
		RemainingMins  int
		UsedMins       int
		DailyLimitMins int
		ExtensionMins  int
		Enabled        bool
		PercentUsed    int
	}

	var userData []UserData
	for _, user := range users {
		remaining, _ := s.store.GetRemainingMinutes(user.ID)
		usedSecs, _ := s.store.GetTodayUsageSeconds(user.ID)
		extensions, _ := s.store.GetTodayExtensions(user.ID)

		totalLimit := user.DailyLimitMins + extensions
		percentUsed := 0
		if totalLimit > 0 {
			percentUsed = (usedSecs / 60) * 100 / totalLimit
			if percentUsed > 100 {
				percentUsed = 100
			}
		}

		userData = append(userData, UserData{
			ID:             user.ID,
			Username:       user.Username,
			IsLoggedIn:     loggedIn[user.Username],
			RemainingMins:  remaining,
			UsedMins:       usedSecs / 60,
			DailyLimitMins: user.DailyLimitMins,
			ExtensionMins:  extensions,
			Enabled:        user.Enabled,
			PercentUsed:    percentUsed,
		})
	}

	data := map[string]interface{}{
		"Title":      "Dashboard",
		"Users":      userData,
		"Now":        time.Now().Format("15:04"),
		"NeedsSetup": s.config.AdminPassword == "",
	}

	s.tmpl.ExecuteTemplate(w, "dashboard.html", data)
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "Manage Users",
		"Users": users,
	}

	s.tmpl.ExecuteTemplate(w, "users.html", data)
}

func (s *Server) handleUserDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByID(id)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	history, err := s.store.GetUsageHistory(id, 7)
	if err != nil {
		history = nil
	}

	remaining, _ := s.store.GetRemainingMinutes(id)
	usedSecs, _ := s.store.GetTodayUsageSeconds(id)
	extensions, _ := s.store.GetTodayExtensions(id)

	data := map[string]interface{}{
		"Title":         user.Username,
		"User":          user,
		"History":       history,
		"RemainingMins": remaining,
		"UsedMins":      usedSecs / 60,
		"ExtensionMins": extensions,
	}

	s.tmpl.ExecuteTemplate(w, "user_detail.html", data)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type UserHistory struct {
		Username string
		History  []*storage.UsageRecord
	}

	var allHistory []UserHistory
	for _, user := range users {
		history, _ := s.store.GetUsageHistory(user.ID, 7)
		allHistory = append(allHistory, UserHistory{
			Username: user.Username,
			History:  history,
		})
	}

	data := map[string]interface{}{
		"Title":   "Usage History",
		"History": allHistory,
	}

	s.tmpl.ExecuteTemplate(w, "history.html", data)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "Settings",
		"Config": s.config,
	}

	s.tmpl.ExecuteTemplate(w, "settings.html", data)
}

// --- HTMX partials ---

func (s *Server) handleStatusPartial(w http.ResponseWriter, r *http.Request) {
	s.handleDashboard(w, r) // Reuse dashboard for now
}

func (s *Server) handleUsersPartial(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.tmpl.ExecuteTemplate(w, "users_partial.html", users)
}

// --- API endpoints ---

func (s *Server) apiGetStatus(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessions, _ := s.logind.ListSessions()
	loggedIn := make(map[string]bool)
	for _, session := range sessions {
		loggedIn[session.UserName] = true
	}

	type Status struct {
		Username      string `json:"username"`
		IsLoggedIn    bool   `json:"is_logged_in"`
		RemainingMins int    `json:"remaining_mins"`
		UsedMins      int    `json:"used_mins"`
		LimitMins     int    `json:"limit_mins"`
		ExtensionMins int    `json:"extension_mins"`
		Enabled       bool   `json:"enabled"`
	}

	var statuses []Status
	for _, user := range users {
		remaining, _ := s.store.GetRemainingMinutes(user.ID)
		usedSecs, _ := s.store.GetTodayUsageSeconds(user.ID)
		extensions, _ := s.store.GetTodayExtensions(user.ID)

		statuses = append(statuses, Status{
			Username:      user.Username,
			IsLoggedIn:    loggedIn[user.Username],
			RemainingMins: remaining,
			UsedMins:      usedSecs / 60,
			LimitMins:     user.DailyLimitMins,
			ExtensionMins: extensions,
			Enabled:       user.Enabled,
		})
	}

	jsonResponse(w, statuses)
}

func (s *Server) apiCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username       string `json:"username"`
		DailyLimitMins int    `json:"daily_limit_mins"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		jsonError(w, "Username is required", http.StatusBadRequest)
		return
	}

	if req.DailyLimitMins <= 0 {
		req.DailyLimitMins = 120 // Default 2 hours
	}

	user, err := s.store.CreateUser(req.Username, req.DailyLimitMins)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, user)
}

func (s *Server) apiUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		DailyLimitMins int  `json:"daily_limit_mins"`
		Enabled        bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateUser(id, req.DailyLimitMins, req.Enabled); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "updated"})
}

func (s *Server) apiDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteUser(id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "deleted"})
}

func (s *Server) apiExtendTime(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Minutes int `json:"minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Minutes <= 0 {
		jsonError(w, "Minutes must be positive", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByID(id)
	if err != nil || user == nil {
		jsonError(w, "User not found", http.StatusNotFound)
		return
	}

	if err := s.store.AddTimeExtension(id, req.Minutes, "parent"); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify the user
	ctx := context.Background()
	s.notifier.SendTimeExtended(ctx, user.Username, req.Minutes)

	remaining, _ := s.store.GetRemainingMinutes(id)
	jsonResponse(w, map[string]interface{}{
		"status":         "extended",
		"minutes_added":  req.Minutes,
		"remaining_mins": remaining,
	})
}

func (s *Server) apiLockUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByID(id)
	if err != nil || user == nil {
		jsonError(w, "User not found", http.StatusNotFound)
		return
	}

	if err := s.logind.LockUserSessions(user.Username); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "locked"})
}

func (s *Server) apiUnlockUser(w http.ResponseWriter, r *http.Request) {
	// Note: There's no direct "unlock" in logind - user needs to enter password
	// This could potentially add bonus time or reset warnings
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByID(id)
	if err != nil || user == nil {
		jsonError(w, "User not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]string{
		"status":  "info",
		"message": "User must unlock screen with their password. Consider extending their time if needed.",
	})
}

// --- Helpers ---

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
