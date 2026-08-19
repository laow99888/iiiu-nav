package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

var ErrIncompatibleSchema = errors.New("restore database schema is not supported")

func PrepareRestoreCandidate(ctx context.Context, databasePath string, maximumSchema int) (int, error) {
	absolute, err := filepath.Abs(databasePath)
	if err != nil {
		return 0, fmt.Errorf("resolve restore database: %w", err)
	}
	uriPath := filepath.ToSlash(absolute)
	if filepath.VolumeName(absolute) != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	dsn := &url.URL{Scheme: "file", Path: uriPath}
	query := dsn.Query()
	query.Set("_foreign_keys", "on")
	query.Set("_busy_timeout", "5000")
	dsn.RawQuery = query.Encode()
	database, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return 0, fmt.Errorf("open restore database: %w", err)
	}
	database.SetMaxOpenConns(1)
	defer database.Close()
	var integrity string
	if err := database.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return 0, fmt.Errorf("check restore database integrity: %w", err)
	}
	if integrity != "ok" {
		return 0, fmt.Errorf("restore database integrity check failed: %s", integrity)
	}
	var schemaVersion int
	if err := database.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&schemaVersion); err != nil {
		return 0, fmt.Errorf("read restore schema version: %w", err)
	}
	if schemaVersion <= 0 || schemaVersion > maximumSchema {
		return 0, ErrIncompatibleSchema
	}
	var adminCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin WHERE id = 1`).Scan(&adminCount); err != nil || adminCount != 1 {
		return 0, errors.New("restore database has no administrator")
	}
	if _, err := database.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return 0, fmt.Errorf("invalidate restored sessions: %w", err)
	}
	if _, err := database.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return 0, fmt.Errorf("checkpoint restore database: %w", err)
	}
	if _, err := database.ExecContext(ctx, `PRAGMA journal_mode = DELETE`); err != nil {
		return 0, fmt.Errorf("finalize restore database: %w", err)
	}
	return schemaVersion, nil
}
