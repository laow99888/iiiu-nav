package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"iiiu-nav/internal/auth"
)

func TestBackupSnapshotIsConsistentDuringWALWritesAndDropsSessions(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()
	if _, err := store.UpsertAdmin(ctx, "test-password-hash"); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC()
	if err := store.CreateSession(ctx, auth.Session{TokenHash: make([]byte, 32), CreatedAt: created, ExpiresAt: created.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var writers sync.WaitGroup
	writers.Add(1)
	go func() {
		defer writers.Done()
		for index := 0; index < 30; index++ {
			value, _ := json.Marshal(map[string]int{"value": index})
			if err := store.PutSetting(ctx, "live-write", value); err != nil {
				t.Errorf("live WAL write: %v", err)
				return
			}
		}
	}()
	snapshotPath := filepath.Join(t.TempDir(), "snapshot.db")
	version, err := store.BackupSnapshot(ctx, snapshotPath)
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	writers.Wait()
	if version != LatestSchemaVersion {
		t.Fatalf("unexpected schema version %d", version)
	}
	snapshot, err := sql.Open("sqlite", snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	var sessions int
	if err := snapshot.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("snapshot retained sessions: count=%d err=%v", sessions, err)
	}
	var integrity string
	if err := snapshot.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("snapshot is not integral: %q %v", integrity, err)
	}
}
