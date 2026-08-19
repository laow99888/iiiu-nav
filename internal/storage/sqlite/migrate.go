package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const LatestSchemaVersion = 2

var (
	ErrNewerSchema       = errors.New("database schema is newer than this application")
	ErrUnversionedSchema = errors.New("database has tables but no migration history")
)

type SchemaState struct {
	Exists         bool
	Version        int
	NeedsMigration bool
}

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int
	name    string
	path    string
}

var migrations = []migration{
	{version: 1, name: "initial", path: "migrations/001_initial.sql"},
	{version: 2, name: "link_url_index", path: "migrations/002_link_url_index.sql"},
}

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at INTEGER NOT NULL
) STRICT;
`

func applyMigrations(ctx context.Context, database *sql.DB) error {
	return applyMigrationSet(ctx, database, migrationFiles, migrations, LatestSchemaVersion)
}

type migrationQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func appliedMigrations(ctx context.Context, querier migrationQuerier, latest int) (map[int]string, error) {
	rows, err := querier.QueryContext(ctx, `SELECT version, name FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read schema migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]string)
	for rows.Next() {
		var version int
		var name string
		if err := rows.Scan(&version, &name); err != nil {
			return nil, fmt.Errorf("scan schema migration: %w", err)
		}
		if version < 1 {
			return nil, fmt.Errorf("database schema migration version %d is invalid", version)
		}
		if version > latest {
			return nil, newerSchemaError(version, latest)
		}
		applied[version] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema migrations: %w", err)
	}

	for version := 1; version <= len(applied); version++ {
		if _, ok := applied[version]; !ok {
			return nil, fmt.Errorf("database schema migration history has a gap at version %d", version)
		}
	}

	return applied, nil
}

func applyMigrationSet(ctx context.Context, database *sql.DB, files fs.FS, set []migration, latest int) error {
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migrations: %w", err)
	}
	defer transaction.Rollback()

	if _, err := transaction.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	applied, err := appliedMigrations(ctx, transaction, latest)
	if err != nil {
		return err
	}
	for _, migration := range set {
		if appliedName, ok := applied[migration.version]; ok {
			if appliedName != migration.name {
				return fmt.Errorf("schema migration %d name mismatch: database has %q, binary expects %q", migration.version, appliedName, migration.name)
			}
			continue
		}
		script, err := fs.ReadFile(files, migration.path)
		if err != nil {
			return fmt.Errorf("read schema migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err := transaction.ExecContext(ctx, string(script)); err != nil {
			return fmt.Errorf("apply schema migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			migration.version,
			migration.name,
			time.Now().UTC().UnixMilli(),
		); err != nil {
			return fmt.Errorf("record schema migration %d (%s): %w", migration.version, migration.name, err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit schema migrations: %w", err)
	}
	return nil
}

func InspectSchema(ctx context.Context, databasePath string) (SchemaState, error) {
	info, err := os.Stat(databasePath)
	if errors.Is(err, os.ErrNotExist) {
		return SchemaState{NeedsMigration: true}, nil
	}
	if err != nil {
		return SchemaState{}, fmt.Errorf("inspect sqlite database: %w", err)
	}
	if !info.Mode().IsRegular() {
		return SchemaState{}, errors.New("sqlite database path is not a regular file")
	}
	if info.Size() == 0 {
		return SchemaState{NeedsMigration: true}, nil
	}

	dsn, err := readOnlyDataSourceName(databasePath)
	if err != nil {
		return SchemaState{}, err
	}
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return SchemaState{}, fmt.Errorf("open sqlite database for schema inspection: %w", err)
	}
	database.SetMaxOpenConns(1)
	defer database.Close()

	var hasHistory bool
	if err := database.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type = 'table' AND name = 'schema_migrations')`).Scan(&hasHistory); err != nil {
		return SchemaState{}, fmt.Errorf("inspect schema migration history: %w", err)
	}
	if !hasHistory {
		var tableCount int
		if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tableCount); err != nil {
			return SchemaState{}, fmt.Errorf("inspect unversioned schema: %w", err)
		}
		if tableCount > 0 {
			return SchemaState{}, fmt.Errorf("%w; restore a compatible backup or start with an empty data directory", ErrUnversionedSchema)
		}
		return SchemaState{NeedsMigration: true}, nil
	}
	applied, err := appliedMigrations(ctx, database, LatestSchemaVersion)
	if err != nil {
		return SchemaState{}, err
	}
	for _, migration := range migrations {
		if name, ok := applied[migration.version]; ok && name != migration.name {
			return SchemaState{}, fmt.Errorf("schema migration %d name mismatch: database has %q, binary expects %q", migration.version, name, migration.name)
		}
	}
	version := len(applied)
	return SchemaState{Exists: true, Version: version, NeedsMigration: version < LatestSchemaVersion}, nil
}

func newerSchemaError(version, latest int) error {
	return fmt.Errorf("%w: database is version %d but this binary supports up to %d; use a binary that supports version %d, or restore a compatible backup before rolling back", ErrNewerSchema, version, latest, version)
}

func readOnlyDataSourceName(databasePath string) (string, error) {
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
	query.Set("mode", "ro")
	dsn.RawQuery = query.Encode()
	return dsn.String(), nil
}
