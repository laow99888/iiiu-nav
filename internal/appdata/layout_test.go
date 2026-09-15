package appdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareCreatesPersistentLayout(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "application data")
	layout, err := Prepare(root)
	if err != nil {
		t.Fatalf("prepare layout: %v", err)
	}

	wantDatabase := filepath.Join(root, "nav.db")
	if layout.Database != wantDatabase {
		t.Fatalf("expected database path %q, got %q", wantDatabase, layout.Database)
	}

	wantTemp := filepath.Join(root, "tmp")
	if layout.Temp != wantTemp {
		t.Fatalf("expected temp path %q, got %q", wantTemp, layout.Temp)
	}

	for _, directory := range []string{
		layout.Root,
		layout.Uploads,
		layout.Logos,
		layout.Backgrounds,
		layout.Site,
		layout.Backups,
		layout.Temp,
	} {
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatalf("stat %q: %v", directory, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", directory)
		}
	}
}

func TestPrepareRejectsInvalidRoot(t *testing.T) {
	t.Parallel()

	if _, err := Prepare(""); err == nil {
		t.Fatal("expected an empty root to fail")
	}

	rootFile := filepath.Join(t.TempDir(), "data")
	if err := os.WriteFile(rootFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create root file: %v", err)
	}
	if _, err := Prepare(rootFile); err == nil {
		t.Fatal("expected a file root to fail")
	}
}
