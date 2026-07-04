package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gaia-calendar/ent"
	"gaia-calendar/ent/appsession"
	"gaia-calendar/internal/config"
	"gaia-calendar/internal/email"
	"gaia-calendar/internal/gaia"
	"gaia-calendar/internal/security"
)

type Server struct {
	cfg       config.Config
	db        *ent.Client
	encryptor security.Encryptor
	emailer   email.CloudflareSender
	gaia      gaia.Client
	mux       *http.ServeMux
	syncMu    sync.Mutex
}

type ctxKey string

const userKey ctxKey = "user"

func New(cfg config.Config, db *ent.Client) *Server {
	s := &Server{
		cfg:       cfg,
		db:        db,
		encryptor: security.NewEncryptor(cfg.CredentialEncryptionKey),
		emailer: email.CloudflareSender{
			AccountID: cfg.CloudflareAccountID,
			APIToken:  cfg.CloudflareAPIToken,
			From:      cfg.CloudflareFrom,
		},
		gaia: gaia.NewClient(),
		mux:  http.NewServeMux(),
	}
	s.routes()
	s.startScheduleSyncLoop()
	return s
}

func (s *Server) startScheduleSyncLoop() {
	go func() {
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		<-timer.C
		s.syncAllGaiaUsers(context.Background())

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.syncAllGaiaUsers(context.Background())
		}
	}()
}

func (s *Server) syncAllGaiaUsers(ctx context.Context) {
	if !s.syncMu.TryLock() {
		return
	}
	defer s.syncMu.Unlock()

	credentials, err := s.db.GaiaCredential.Query().WithUser().All(ctx)
	if err != nil {
		slog.Warn("scheduled Gaia sync cannot load credentials", "error", err)
		return
	}
	for _, cred := range credentials {
		if cred.Edges.User == nil {
			continue
		}
		userCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		_, err := s.syncSchedulesForUser(userCtx, cred.Edges.User)
		cancel()
		if err != nil {
			slog.Warn("scheduled Gaia sync failed", "userID", cred.Edges.User.ID, "error", err)
		}
	}
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/config", s.handlePublicConfig)
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/verify", s.handleVerify)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/request-password-reset", s.handleRequestPasswordReset)
	s.mux.HandleFunc("POST /api/auth/reset-password", s.handleResetPassword)
	s.mux.Handle("POST /api/auth/logout", s.auth(http.HandlerFunc(s.handleLogout)))
	s.mux.Handle("GET /api/me", s.auth(http.HandlerFunc(s.handleMe)))
	s.mux.Handle("GET /api/admin/users", s.auth(s.admin(http.HandlerFunc(s.handleAdminUsers))))
	s.mux.Handle("PATCH /api/admin/users/", s.auth(s.admin(http.HandlerFunc(s.handleAdminUpdateUser))))
	s.mux.Handle("GET /api/gaia-credential", s.auth(http.HandlerFunc(s.handleGetGaiaCredential)))
	s.mux.Handle("PUT /api/gaia-credential", s.auth(http.HandlerFunc(s.handleSaveGaiaCredential)))
	s.mux.Handle("GET /api/calendar-subscription", s.auth(http.HandlerFunc(s.handleGetCalendarSubscription)))
	s.mux.Handle("POST /api/calendar-subscription/rotate", s.auth(http.HandlerFunc(s.handleRotateCalendarSubscription)))
	s.mux.Handle("GET /api/calendar-request-logs", s.auth(http.HandlerFunc(s.handleGetCalendarRequestLogs)))
	s.mux.Handle("POST /api/schedules/sync", s.auth(http.HandlerFunc(s.handleSyncSchedules)))
	s.mux.Handle("GET /api/schedules", s.auth(http.HandlerFunc(s.handleSchedules)))
	s.mux.Handle("GET /api/sync-runs", s.auth(http.HandlerFunc(s.handleSyncRuns)))
	s.mux.Handle("PATCH /api/sync-runs/", s.auth(http.HandlerFunc(s.handleUpdateSyncRun)))
	s.mux.Handle("DELETE /api/sync-runs/", s.auth(http.HandlerFunc(s.handleDeleteSyncRun)))
	s.mux.HandleFunc("GET /calendar/", s.handleCalendarFeed)
	s.mux.HandleFunc("/", s.handleFrontend)
}

func (s *Server) handlePublicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, publicConfigResponse{
		EmailVerificationRequired: s.cfg.EmailVerificationRequired,
	})
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("gaia_calendar_session")
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		hash := security.HashToken(cookie.Value)
		sess, err := s.db.AppSession.Query().
			Where(appsession.TokenHash(hash), appsession.ExpiresAtGT(time.Now()), appsession.RevokedAtIsNil()).
			WithUser().
			Only(r.Context())
		if err != nil || sess.Edges.User == nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, sess.Edges.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func currentUser(r *http.Request) *ent.User {
	u, _ := r.Context().Value(userKey).(*ent.User)
	return u
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func userDTO(u *ent.User) userResponse {
	nickname := ""
	if u.Nickname != nil {
		nickname = *u.Nickname
	}
	return userResponse{ID: u.ID, Email: u.Email, Nickname: nickname, EmailVerified: u.EmailVerified, Role: u.Role}
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "gaia_calendar_session",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.BaseURL, "https://"),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "gaia_calendar_session", Value: "", Path: "/", Expires: time.Unix(0, 0), HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "." || path == "/" {
		path = "index.html"
	}
	full := filepath.Join(s.cfg.FrontendDir, strings.TrimPrefix(path, "/"))
	if _, err := os.Stat(full); err == nil {
		http.ServeFile(w, r, full)
		return
	}
	index := filepath.Join(s.cfg.FrontendDir, "index.html")
	if _, err := os.Stat(index); err == nil {
		http.ServeFile(w, r, index)
		return
	}
	writeError(w, http.StatusNotFound, "frontend is not built")
}

func normalizeEmail(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func validPassword(v string) bool {
	return len(v) >= 8
}

func normalizeNickname(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func validNickname(v string) bool {
	if len(v) < 3 || len(v) > 32 {
		return false
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func logHTTPError(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("request failed", "error", err)
	}
}
