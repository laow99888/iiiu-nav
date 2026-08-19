package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"iiiu-nav/internal/appdata"
	"iiiu-nav/internal/backup"
	storage "iiiu-nav/internal/storage/sqlite"
)

func TestOpenPreparedApplicationDataBacksUpBeforeMigration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	layout, err := appdata.Prepare(t.TempDir())
	if err != nil {
		t.Fatalf("prepare data layout: %v", err)
	}
	makeSchemaOneDatabase(t, ctx, layout.Database)

	data, err := openPreparedApplicationData(ctx, layout, "upgrade-test")
	if err != nil {
		t.Fatalf("open application data: %v", err)
	}
	defer data.Close()
	if data.upgrade.FromVersion != 1 || data.upgrade.ToVersion != storage.LatestSchemaVersion || data.upgrade.Backup == nil {
		t.Fatalf("unexpected upgrade result: %+v", data.upgrade)
	}
	version, err := data.store.SchemaVersion(ctx)
	if err != nil || version != storage.LatestSchemaVersion {
		t.Fatalf("expected migrated schema %d, got %d: %v", storage.LatestSchemaVersion, version, err)
	}

	manifest := readBackupManifest(t, filepath.Join(layout.Backups, data.upgrade.Backup.Name))
	if manifest.SchemaVersion != 1 || manifest.ApplicationVersion != "upgrade-test" {
		t.Fatalf("unexpected pre-migration manifest: %+v", manifest)
	}
}

func TestOpenPreparedApplicationDataRejectsNewerSchemaBeforeBackup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	layout, err := appdata.Prepare(t.TempDir())
	if err != nil {
		t.Fatalf("prepare data layout: %v", err)
	}
	store, err := storage.Open(ctx, layout.Database)
	if err != nil {
		t.Fatalf("create current database: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close current database: %v", err)
	}
	database := openRawDatabase(t, layout.Database)
	if _, err := database.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, 'future', 0)`, storage.LatestSchemaVersion+1); err != nil {
		t.Fatalf("insert future migration: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close future database: %v", err)
	}

	data, err := openPreparedApplicationData(ctx, layout, "old-binary")
	if data != nil {
		_ = data.Close()
		t.Fatal("expected startup to reject a newer schema")
	}
	if !errors.Is(err, storage.ErrNewerSchema) {
		t.Fatalf("expected newer schema error, got %v", err)
	}
	entries, err := os.ReadDir(layout.Backups)
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("newer schema should not be modified or backed up, got %d files", len(entries))
	}
}

func TestOpenPreparedApplicationDataPreservesBackupAndDatabaseWhenMigrationFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	layout, err := appdata.Prepare(t.TempDir())
	if err != nil {
		t.Fatalf("prepare data layout: %v", err)
	}
	makeSchemaOneDatabase(t, ctx, layout.Database)
	database := openRawDatabase(t, layout.Database)
	if _, err := database.ExecContext(ctx, `CREATE INDEX links_url_idx ON links (name)`); err != nil {
		t.Fatalf("create conflicting index: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close conflicting database: %v", err)
	}

	data, err := openPreparedApplicationData(ctx, layout, "failing-upgrade")
	if data != nil {
		_ = data.Close()
		t.Fatal("expected startup to abort after migration failure")
	}
	if err == nil || !strings.Contains(err.Error(), "migration 2 (link_url_index)") || !strings.Contains(err.Error(), "pre-migration backup") {
		t.Fatalf("unexpected migration error: %v", err)
	}

	database = openRawDatabase(t, layout.Database)
	defer database.Close()
	var version int
	if err := database.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read preserved schema: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected failed upgrade to preserve schema 1, got %d", version)
	}
	entries, err := os.ReadDir(layout.Backups)
	if err != nil {
		t.Fatalf("list preserved backups: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one preserved pre-migration backup, got %d", len(entries))
	}
	manifest := readBackupManifest(t, filepath.Join(layout.Backups, entries[0].Name()))
	if manifest.SchemaVersion != 1 {
		t.Fatalf("expected schema-one backup, got schema %d", manifest.SchemaVersion)
	}
}

func makeSchemaOneDatabase(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	store, err := storage.Open(ctx, path)
	if err != nil {
		t.Fatalf("create current database: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close current database: %v", err)
	}
	database := openRawDatabase(t, path)
	if _, err := database.ExecContext(ctx, `DROP INDEX links_url_idx`); err != nil {
		t.Fatalf("remove v2 index: %v", err)
	}
	if _, err := database.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = 2`); err != nil {
		t.Fatalf("remove v2 migration record: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close schema-one database: %v", err)
	}
}

func openRawDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw database: %v", err)
	}
	database.SetMaxOpenConns(1)
	return database
}

func readBackupManifest(t *testing.T, path string) backup.Manifest {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open backup archive: %v", err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Name != "manifest.json" {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			t.Fatalf("open backup manifest: %v", err)
		}
		defer file.Close()
		var manifest backup.Manifest
		if err := json.NewDecoder(file).Decode(&manifest); err != nil {
			t.Fatalf("decode backup manifest: %v", err)
		}
		return manifest
	}
	t.Fatal("backup manifest not found")
	return backup.Manifest{}
}
