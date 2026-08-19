package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	database     *sql.DB
	databasePath string
}

func Open(ctx context.Context, databasePath string) (*Store, error) {
	return openStore(ctx, databasePath, true)
}

// OpenForBackup opens a preflighted database without changing its schema.
func OpenForBackup(ctx context.Context, databasePath string) (*Store, error) {
	return openStore(ctx, databasePath, false)
}

func openStore(ctx context.Context, databasePath string, migrate bool) (*Store, error) {
	dsn, err := dataSourceName(databasePath)
	if err != nil {
		return nil, err
	}

	database, err := openDatabase(ctx, dsn, migrate)
	if err != nil {
		return nil, err
	}
	return &Store{database: database, databasePath: databasePath}, nil
}

func openDatabase(ctx context.Context, dsn string, migrate bool) (*sql.DB, error) {
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("connect to sqlite database: %w", err)
	}
	if err := validatePragmas(ctx, database); err != nil {
		_ = database.Close()
		return nil, err
	}
	if migrate {
		if err := applyMigrations(ctx, database); err != nil {
			_ = database.Close()
			return nil, err
		}
	}
	return database, nil
}

func (store *Store) Close() error {
	if store == nil || store.database == nil {
		return nil
	}
	if err := store.database.Close(); err != nil {
		return fmt.Errorf("close sqlite database: %w", err)
	}
	return nil
}

func (store *Store) Reopen(ctx context.Context) error {
	dsn, err := dataSourceName(store.databasePath)
	if err != nil {
		return err
	}
	database, err := openDatabase(ctx, dsn, true)
	if err != nil {
		return err
	}
	store.database = database
	return nil
}

func (store *Store) SchemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := store.database.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func dataSourceName(databasePath string) (string, error) {
	if databasePath == "" {
		return "", errors.New("database path is required")
	}

	absolutePath, err := filepath.Abs(databasePath)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}
	uriPath := filepath.ToSlash(absolutePath)
	if filepath.VolumeName(absolutePath) != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}

	dsn := &url.URL{Scheme: "file", Path: uriPath}
	query := dsn.Query()
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "NORMAL")
	dsn.RawQuery = query.Encode()

	return dsn.String(), nil
}

func validatePragmas(ctx context.Context, database *sql.DB) error {
	checks := []struct {
		name string
		want string
	}{
		{name: "foreign_keys", want: "1"},
		{name: "journal_mode", want: "wal"},
		{name: "synchronous", want: "1"},
		{name: "busy_timeout", want: "5000"},
	}

	for _, check := range checks {
		var value string
		if err := database.QueryRowContext(ctx, "PRAGMA "+check.name).Scan(&value); err != nil {
			return fmt.Errorf("read sqlite pragma %s: %w", check.name, err)
		}
		if !strings.EqualFold(value, check.want) {
			return fmt.Errorf("sqlite pragma %s is %q, expected %q", check.name, value, check.want)
		}
	}

	return nil
}
