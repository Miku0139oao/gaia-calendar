package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gaia-calendar/internal/config"
	"gaia-calendar/internal/security"
)

func TestSyncRunsSupportLimitAndOffset(t *testing.T) {
	db := openHTTPAPITestDB(t, "sync-runs-pagination.db")
	defer db.Close()
	u := db.User.Create().
		SetEmail("runs@example.com").
		SetPasswordHash("hash").
		SetEmailVerified(true).
		SaveX(t.Context())
	for i := 0; i < 25; i++ {
		started := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute)
		db.ScheduleSyncRun.Create().
			SetUser(u).
			SetStartedAt(started).
			SetRangeStart(started).
			SetRangeEnd(started).
			SetStatus("success").
			SaveX(t.Context())
	}
	sessionToken := "runs-session"
	db.AppSession.Create().
		SetUser(u).
		SetTokenHash(security.HashToken(sessionToken)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SaveX(t.Context())
	server := New(config.Config{
		BaseURL:                   "http://localhost:8080",
		SessionSecret:             "dev-session-secret-change-me",
		CredentialEncryptionKey:   "dev-encryption-secret-change-me",
		EmailVerificationRequired: false,
	}, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sync-runs?limit=10&offset=10", nil)
	req.AddCookie(&http.Cookie{Name: "gaia_calendar_session", Value: sessionToken})
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("sync runs status = %d, body = %s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, want := range []string{`"total":25`, `"limit":10`, `"offset":10`, `"hasMore":true`} {
		if !strings.Contains(body, want) {
			t.Fatalf("sync runs response missing %q: %s", want, body)
		}
	}
	if got := strings.Count(body, `"id":`); got != 10 {
		t.Fatalf("sync runs returned %d rows, want 10: %s", got, body)
	}
}
