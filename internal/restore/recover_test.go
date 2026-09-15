package restore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoverInterruptedRestoresPreviousDataFromRollback(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rollback := filepath.Join(root, rollbackPrefix+"123")
	if err := os.MkdirAll(filepath.Join(rollback, uploadsDirName, "logos"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rollback, databaseFileName), []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rollback, databaseFileName+"-wal"), []byte("wal"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rollback, uploadsDirName, "logos", "a.png"), []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Crash point: the active database was moved away and its companions were
	// not; appdata.Prepare also recreated an empty uploads directory.
	if err := os.WriteFile(filepath.Join(root, databaseFileName+"-shm"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, uploadsDirName), 0o700); err != nil {
		t.Fatal(err)
	}

	actions, err := RecoverInterrupted(root)
	if err != nil {
		t.Fatalf("recover interrupted restore: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected recovery actions to be reported")
	}

	database, err := os.ReadFile(filepath.Join(root, databaseFileName))
	if err != nil || string(database) != "previous" {
		t.Fatalf("expected the previous database to be restored, got %q (err=%v)", database, err)
	}
	wal, err := os.ReadFile(filepath.Join(root, databaseFileName+"-wal"))
	if err != nil || string(wal) != "wal" {
		t.Fatalf("expected the previous write-ahead log to be restored, got %q (err=%v)", wal, err)
	}
	if _, err := os.Stat(filepath.Join(root, databaseFileName+"-shm")); !os.IsNotExist(err) {
		t.Fatalf("expected the stale shared-memory file to be removed with the rollback workspace, got %v", err)
	}
	uploaded, err := os.ReadFile(filepath.Join(root, uploadsDirName, "logos", "a.png"))
	if err != nil || string(uploaded) != "png" {
		t.Fatalf("expected the previous uploads to be restored: %v", err)
	}
	if _, err := os.Stat(rollback); !os.IsNotExist(err) {
		t.Fatalf("expected the rollback directory to be removed, got %v", err)
	}
}

func TestRecoverInterruptedKeepsRollbackWhenDatabaseIsActive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, databaseFileName), []byte("active"), 0o600); err != nil {
		t.Fatal(err)
	}
	rollback := filepath.Join(root, rollbackPrefix+"456")
	if err := os.MkdirAll(rollback, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rollback, databaseFileName), []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}

	actions, err := RecoverInterrupted(root)
	if err != nil {
		t.Fatalf("recover interrupted restore: %v", err)
	}
	database, err := os.ReadFile(filepath.Join(root, databaseFileName))
	if err != nil || string(database) != "active" {
		t.Fatalf("expected the active database to stay untouched, got %q (err=%v)", database, err)
	}
	if _, err := os.Stat(filepath.Join(rollback, databaseFileName)); err != nil {
		t.Fatalf("expected the rollback copy to be kept for the operator: %v", err)
	}
	if len(actions) != 1 || !strings.Contains(actions[0], rollbackPrefix+"456") {
		t.Fatalf("expected a leftover notice for the rollback directory, got %v", actions)
	}
}

func TestRecoverInterruptedRemovesValidatingWorkspacesAndEmptyRollbacks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	validating := filepath.Join(root, validatingPrefix+"789")
	if err := os.MkdirAll(filepath.Join(validating, uploadsDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(validating, databaseFileName), []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyRollback := filepath.Join(root, rollbackPrefix+"000")
	if err := os.MkdirAll(emptyRollback, 0o700); err != nil {
		t.Fatal(err)
	}

	actions, err := RecoverInterrupted(root)
	if err != nil {
		t.Fatalf("recover interrupted restore: %v", err)
	}
	for _, directory := range []string{validating, emptyRollback} {
		if _, err := os.Stat(directory); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, got %v", directory, err)
		}
	}
	if len(actions) != 2 {
		t.Fatalf("expected two cleanup actions, got %v", actions)
	}
}

func TestRecoverInterruptedLeavesCleanRootUntouched(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, uploadsDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, databaseFileName), []byte("active"), 0o600); err != nil {
		t.Fatal(err)
	}

	actions, err := RecoverInterrupted(root)
	if err != nil {
		t.Fatalf("recover interrupted restore: %v", err)
	}
	if len(actions) != 0 {
		t.Fatalf("expected no actions on a clean root, got %v", actions)
	}
}
