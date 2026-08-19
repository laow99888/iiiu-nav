package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestOpenCreatesConfiguredSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)

	version, err := store.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != LatestSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", LatestSchemaVersion, version)
	}

	pragmas := map[string]string{
		"foreign_keys": "1",
		"journal_mode": "wal",
		"synchronous":  "1",
		"busy_timeout": "5000",
	}
	for name, want := range pragmas {
		var got string
		if err := store.database.QueryRowContext(ctx, "PRAGMA "+name).Scan(&got); err != nil {
			t.Fatalf("read pragma %s: %v", name, err)
		}
		if !strings.EqualFold(got, want) {
			t.Errorf("expected pragma %s=%s, got %s", name, want, got)
		}
	}

	wantTables := map[string]bool{
		"admin":             false,
		"categories":        false,
		"daily_page_views":  false,
		"links":             false,
		"schema_migrations": false,
		"sessions":          false,
		"settings":          false,
	}
	var indexCount int
	if err := store.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'index' AND name = 'links_url_idx'`).Scan(&indexCount); err != nil {
		t.Fatalf("inspect v2 index: %v", err)
	}
	if indexCount != 1 {
		t.Fatal("expected links_url_idx to be created")
	}
	rows, err := store.database.QueryContext(ctx, `SELECT name FROM sqlite_schema WHERE type = 'table'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		if _, ok := wantTables[name]; ok {
			wantTables[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}
	for name, found := range wantTables {
		if !found {
			t.Errorf("expected table %q", name)
		}
	}
}

func TestOpenMigratesExistingDatabaseIdempotently(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "existing.db")
	dsn, err := dataSourceName(databasePath)
	if err != nil {
		t.Fatalf("create DSN: %v", err)
	}
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open existing database: %v", err)
	}
	if _, err := database.ExecContext(ctx, createMigrationsTable); err != nil {
		t.Fatalf("create migration metadata: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close existing database: %v", err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		store, err := Open(ctx, databasePath)
		if err != nil {
			t.Fatalf("open attempt %d: %v", attempt, err)
		}
		var migrationCount int
		if err := store.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
			_ = store.Close()
			t.Fatalf("count migrations on attempt %d: %v", attempt, err)
		}
		if migrationCount != LatestSchemaVersion {
			_ = store.Close()
			t.Fatalf("expected %d migration rows, got %d", LatestSchemaVersion, migrationCount)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("close attempt %d: %v", attempt, err)
		}
	}
}

func TestOpenRejectsNewerSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "future.db")
	dsn, err := dataSourceName(databasePath)
	if err != nil {
		t.Fatalf("create DSN: %v", err)
	}
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open future database: %v", err)
	}
	if _, err := database.ExecContext(ctx, createMigrationsTable); err != nil {
		t.Fatalf("create migration metadata: %v", err)
	}
	if _, err := database.ExecContext(
		ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, 0)`,
		LatestSchemaVersion+1,
		"future",
	); err != nil {
		t.Fatalf("insert future migration: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close future database: %v", err)
	}

	store, err := Open(ctx, databasePath)
	if err == nil {
		_ = store.Close()
		t.Fatal("expected a newer schema to be rejected")
	}
	if !errors.Is(err, ErrNewerSchema) || !strings.Contains(err.Error(), "restore a compatible backup") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInspectSchemaIsReadOnlyAndReportsUpgradeState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "inspect.db")
	state, err := InspectSchema(ctx, databasePath)
	if err != nil {
		t.Fatalf("inspect missing database: %v", err)
	}
	if state.Exists || state.Version != 0 || !state.NeedsMigration {
		t.Fatalf("unexpected missing database state: %+v", state)
	}
	if _, err := os.Stat(databasePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inspection created the database: %v", err)
	}

	store, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("create current database: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close current database: %v", err)
	}
	state, err = InspectSchema(ctx, databasePath)
	if err != nil {
		t.Fatalf("inspect current database: %v", err)
	}
	if !state.Exists || state.Version != LatestSchemaVersion || state.NeedsMigration {
		t.Fatalf("unexpected current database state: %+v", state)
	}
}

func TestMigrationSetRollsBackAllPendingMigrationsOnFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "rollback.db")
	dsn, err := dataSourceName(databasePath)
	if err != nil {
		t.Fatalf("create DSN: %v", err)
	}
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	database.SetMaxOpenConns(1)
	defer database.Close()
	if err := applyMigrationSet(ctx, database, migrationFiles, migrations[:1], 1); err != nil {
		t.Fatalf("apply baseline migration: %v", err)
	}

	files := fstest.MapFS{
		"migrations/002_ok.sql":   {Data: []byte(`CREATE TABLE migration_should_rollback (id INTEGER PRIMARY KEY) STRICT;`)},
		"migrations/003_fail.sql": {Data: []byte(`CREATE TABL invalid_syntax (id INTEGER);`)},
	}
	set := []migration{
		{version: 1, name: "initial", path: "migrations/unused.sql"},
		{version: 2, name: "ok", path: "migrations/002_ok.sql"},
		{version: 3, name: "intentional_failure", path: "migrations/003_fail.sql"},
	}
	err = applyMigrationSet(ctx, database, files, set, 3)
	if err == nil || !strings.Contains(err.Error(), "migration 3 (intentional_failure)") {
		t.Fatalf("expected named migration failure, got %v", err)
	}

	var version int
	if err := database.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read preserved schema version: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected schema version 1 after rollback, got %d", version)
	}
	var tableCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name = 'migration_should_rollback'`).Scan(&tableCount); err != nil {
		t.Fatalf("inspect rolled-back table: %v", err)
	}
	if tableCount != 0 {
		t.Fatal("successful pending migration was not rolled back")
	}
}
