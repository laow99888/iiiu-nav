package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
)

func (store *Store) BackupSnapshot(ctx context.Context, destination string) (int, error) {
	if _, err := os.Stat(destination); err == nil {
		return 0, fmt.Errorf("backup destination already exists")
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("inspect backup destination: %w", err)
	}
	if _, err := store.database.ExecContext(ctx, `VACUUM INTO ?`, destination); err != nil {
		return 0, fmt.Errorf("snapshot sqlite database: %w", err)
	}
	dsn, err := dataSourceName(destination)
	if err != nil {
		return 0, err
	}
	snapshot, err := sql.Open("sqlite", dsn)
	if err != nil {
		return 0, fmt.Errorf("open database snapshot: %w", err)
	}
	snapshot.SetMaxOpenConns(1)
	defer snapshot.Close()
	if _, err := snapshot.ExecContext(ctx, `PRAGMA journal_mode = DELETE`); err != nil {
		return 0, fmt.Errorf("set snapshot journal mode: %w", err)
	}
	if _, err := snapshot.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return 0, fmt.Errorf("remove snapshot sessions: %w", err)
	}
	if _, err := snapshot.ExecContext(ctx, `VACUUM`); err != nil {
		return 0, fmt.Errorf("compact database snapshot: %w", err)
	}
	var integrity string
	if err := snapshot.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return 0, fmt.Errorf("check database snapshot integrity: %w", err)
	}
	if integrity != "ok" {
		return 0, fmt.Errorf("database snapshot integrity check failed: %s", integrity)
	}
	var schemaVersion int
	if err := snapshot.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&schemaVersion); err != nil {
		return 0, fmt.Errorf("read snapshot schema version: %w", err)
	}
	return schemaVersion, nil
}
