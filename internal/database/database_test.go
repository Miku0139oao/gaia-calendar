package database

import (
	"path/filepath"
	"testing"
)

func TestOpenSupportsSQLiteURL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "gaia-calendar.db")
	client, err := Open(t.Context(), "sqlite://"+dbPath)
	if err != nil {
		t.Fatalf("Open sqlite returned error: %v", err)
	}
	defer client.Close()

	if _, err := client.User.Query().Count(t.Context()); err != nil {
		t.Fatalf("query users after schema creation: %v", err)
	}
}
