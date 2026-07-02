package database

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"gaia-calendar/ent"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func Open(ctx context.Context, databaseURL string) (*ent.Client, error) {
	if isSQLiteURL(databaseURL) {
		return openSQLite(ctx, databaseURL)
	}
	client, err := ent.Open(dialect.Postgres, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func isSQLiteURL(databaseURL string) bool {
	return databaseURL == "" ||
		strings.HasPrefix(databaseURL, "sqlite://") ||
		strings.HasPrefix(databaseURL, "sqlite:") ||
		strings.HasSuffix(databaseURL, ".db") ||
		strings.HasSuffix(databaseURL, ".sqlite") ||
		strings.HasSuffix(databaseURL, ".sqlite3")
}

func openSQLite(ctx context.Context, databaseURL string) (*ent.Client, error) {
	dsn := sqliteDSN(databaseURL)
	if err := ensureSQLiteDir(dsn); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	if err := client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func sqliteDSN(databaseURL string) string {
	if databaseURL == "" {
		return "file:data/gaia-calendar.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	if strings.HasPrefix(databaseURL, "sqlite://") {
		value := strings.TrimPrefix(databaseURL, "sqlite://")
		if strings.HasPrefix(value, "file:") {
			return withSQLitePragmas(value)
		}
		return withSQLitePragmas("file:" + value)
	}
	if strings.HasPrefix(databaseURL, "sqlite:") {
		return withSQLitePragmas(strings.TrimPrefix(databaseURL, "sqlite:"))
	}
	if strings.HasPrefix(databaseURL, "file:") {
		return withSQLitePragmas(databaseURL)
	}
	return withSQLitePragmas("file:" + databaseURL)
}

func withSQLitePragmas(dsn string) string {
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
}

func ensureSQLiteDir(dsn string) error {
	path := strings.TrimPrefix(dsn, "file:")
	if parsed, err := url.Parse(path); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	if index := strings.Index(path, "?"); index >= 0 {
		path = path[:index]
	}
	if path == "" || path == ":memory:" || strings.Contains(path, "mode=memory") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
