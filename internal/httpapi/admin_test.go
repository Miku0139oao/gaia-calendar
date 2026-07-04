package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"gaia-calendar/internal/config"
	"gaia-calendar/internal/security"
)

func TestAdminUsersRequiresAdminRole(t *testing.T) {
	db := openHTTPAPITestDB(t, "admin-requires-role.db")
	defer db.Close()
	u := db.User.Create().
		SetEmail("regular@example.com").
		SetPasswordHash("hash").
		SetEmailVerified(true).
		SetRole("user").
		SaveX(t.Context())
	sessionToken := "regular-session"
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

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "gaia_calendar_session", Value: sessionToken})
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("admin users status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminCanListAndUpdateUsers(t *testing.T) {
	db := openHTTPAPITestDB(t, "admin-users.db")
	defer db.Close()
	admin := db.User.Create().
		SetEmail("admin@example.com").
		SetPasswordHash("hash").
		SetEmailVerified(true).
		SetRole("admin").
		SaveX(t.Context())
	target := db.User.Create().
		SetEmail("target@example.com").
		SetPasswordHash("hash").
		SetEmailVerified(false).
		SetRole("user").
		SaveX(t.Context())
	sessionToken := "admin-session"
	db.AppSession.Create().
		SetUser(admin).
		SetTokenHash(security.HashToken(sessionToken)).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SaveX(t.Context())
	server := New(config.Config{
		BaseURL:                   "http://localhost:8080",
		SessionSecret:             "dev-session-secret-change-me",
		CredentialEncryptionKey:   "dev-encryption-secret-change-me",
		EmailVerificationRequired: false,
	}, db)

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	listReq.AddCookie(&http.Cookie{Name: "gaia_calendar_session", Value: sessionToken})
	listRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list users status = %d, body = %s", listRes.Code, listRes.Body.String())
	}
	if body := listRes.Body.String(); !strings.Contains(body, `"total":2`) || !strings.Contains(body, `"target@example.com"`) {
		t.Fatalf("list users response missing expected data: %s", body)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/admin/users/"+strconv.Itoa(target.ID), bytes.NewBufferString(`{"role":"admin","nickname":"targetnick","emailVerified":true}`))
	updateReq.AddCookie(&http.Cookie{Name: "gaia_calendar_session", Value: sessionToken})
	updateRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRes, updateReq)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update user status = %d, body = %s", updateRes.Code, updateRes.Body.String())
	}
	updated := db.User.GetX(t.Context(), target.ID)
	if updated.Role != "admin" || updated.Nickname == nil || *updated.Nickname != "targetnick" || !updated.EmailVerified {
		t.Fatalf("updated user role=%q nickname=%v verified=%v", updated.Role, updated.Nickname, updated.EmailVerified)
	}
}
