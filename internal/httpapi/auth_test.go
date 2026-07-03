package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gaia-calendar/ent"
	"gaia-calendar/ent/user"
	"gaia-calendar/internal/config"
	"gaia-calendar/internal/database"

	_ "modernc.org/sqlite"
)

func TestBuildPasswordResetURL(t *testing.T) {
	got, err := buildPasswordResetURL("https://example.com/", "token+with/slash")
	if err != nil {
		t.Fatalf("buildPasswordResetURL returned error: %v", err)
	}
	want := "https://example.com/reset-password?token=token%2Bwith%2Fslash"
	if got != want {
		t.Fatalf("reset URL = %q, want %q", got, want)
	}
}

func TestValidateRegisterRequestRequiresMatchingPasswordConfirmation(t *testing.T) {
	req := registerRequest{
		Email:           "user@example.com",
		Password:        "password123",
		ConfirmPassword: "different123",
	}

	got := validateRegisterRequest(normalizeEmail(req.Email), req)
	if got != "password confirmation does not match" {
		t.Fatalf("validation error = %q", got)
	}
}

func TestValidateRegisterRequestAcceptsMatchingPasswordConfirmation(t *testing.T) {
	req := registerRequest{
		Email:           "user@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	}

	if got := validateRegisterRequest(normalizeEmail(req.Email), req); got != "" {
		t.Fatalf("validation error = %q", got)
	}
}

func TestRegisterWithoutEmailVerificationMarksUserVerified(t *testing.T) {
	db := openHTTPAPITestDB(t, "auth-register-no-email.db")
	defer db.Close()
	server := New(config.Config{
		BaseURL:                   "http://localhost:8080",
		SessionSecret:             "dev-session-secret-change-me",
		CredentialEncryptionKey:   "dev-encryption-secret-change-me",
		EmailVerificationRequired: false,
	}, db)

	body := bytes.NewBufferString(`{"email":"demo@example.com","password":"password123","confirmPassword":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", res.Code, res.Body.String())
	}
	u, err := db.User.Query().Where(user.Email("demo@example.com")).Only(t.Context())
	if err != nil {
		t.Fatalf("registered user not found: %v", err)
	}
	if !u.EmailVerified {
		t.Fatal("registered user should be email verified when verification is disabled")
	}

	loginBody := bytes.NewBufferString(`{"email":"demo@example.com","password":"password123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginRes := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginRes.Code, loginRes.Body.String())
	}
}

func TestPublicConfigReportsEmailVerificationRequirement(t *testing.T) {
	db := openHTTPAPITestDB(t, "public-config.db")
	defer db.Close()
	server := New(config.Config{
		BaseURL:                   "http://localhost:8080",
		SessionSecret:             "dev-session-secret-change-me",
		CredentialEncryptionKey:   "dev-encryption-secret-change-me",
		EmailVerificationRequired: false,
	}, db)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	res := httptest.NewRecorder()
	server.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("config status = %d, body = %s", res.Code, res.Body.String())
	}
	var got publicConfigResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode config response: %v", err)
	}
	if got.EmailVerificationRequired {
		t.Fatal("public config should report disabled email verification")
	}
}

func openHTTPAPITestDB(t *testing.T, name string) *ent.Client {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), name)
	db, err := database.Open(t.Context(), "sqlite://"+dbPath)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}
