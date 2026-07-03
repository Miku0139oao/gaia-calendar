package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr                      string
	BaseURL                   string
	DatabaseURL               string
	SessionSecret             string
	CredentialEncryptionKey   string
	EmailVerificationRequired bool
	CloudflareAccountID       string
	CloudflareAPIToken        string
	CloudflareFrom            string
	GaiaDefaultCompanyCode    string
	FrontendDir               string
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Addr:                      env("APP_ADDR", ":8080"),
		BaseURL:                   env("APP_BASE_URL", "http://localhost:8080"),
		DatabaseURL:               env("DATABASE_URL", "sqlite://data/gaia-calendar.db"),
		SessionSecret:             env("SESSION_SECRET", "dev-session-secret-change-me"),
		CredentialEncryptionKey:   env("CREDENTIAL_ENCRYPTION_KEY", "dev-encryption-secret-change-me"),
		EmailVerificationRequired: envBool("AUTH_EMAIL_VERIFICATION_REQUIRED", true),
		CloudflareAccountID:       os.Getenv("CLOUDFLARE_EMAIL_ACCOUNT_ID"),
		CloudflareAPIToken:        os.Getenv("CLOUDFLARE_EMAIL_API_TOKEN"),
		CloudflareFrom:            os.Getenv("CLOUDFLARE_EMAIL_FROM"),
		GaiaDefaultCompanyCode:    env("GAIA_DEFAULT_COMPANY_CODE", ""),
		FrontendDir:               env("FRONTEND_DIR", "frontend/dist"),
	}
}

func (c Config) Validate() error {
	if isLocalBaseURL(c.BaseURL) {
		return nil
	}
	if c.SessionSecret == "dev-session-secret-change-me" {
		return fmt.Errorf("SESSION_SECRET must be changed for non-local APP_BASE_URL")
	}
	if c.CredentialEncryptionKey == "dev-encryption-secret-change-me" {
		return fmt.Errorf("CREDENTIAL_ENCRYPTION_KEY must be changed for non-local APP_BASE_URL")
	}
	return nil
}

func isLocalBaseURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
