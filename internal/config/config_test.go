package config

import "testing"

func TestValidateAllowsLocalDevelopmentDefaults(t *testing.T) {
	cfg := Config{
		BaseURL:                 "http://localhost:8080",
		SessionSecret:           "dev-session-secret-change-me",
		CredentialEncryptionKey: "dev-encryption-secret-change-me",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error for local config: %v", err)
	}
}

func TestValidateRejectsProductionDefaults(t *testing.T) {
	cfg := Config{
		BaseURL:                 "https://calendar.example.com",
		SessionSecret:           "dev-session-secret-change-me",
		CredentialEncryptionKey: "dev-encryption-secret-change-me",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate should reject default secrets for non-local base URL")
	}
}
