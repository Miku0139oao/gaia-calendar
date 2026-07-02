package httpapi

import "testing"

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
