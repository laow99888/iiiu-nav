package backup_test

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/navigation"
	storage "iiiu-nav/internal/storage/sqlite"
)

func TestCreateListOpenAndDeleteFullBackup(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	backupsRoot := filepath.Join(dataRoot, "backups")
	uploadsRoot := filepath.Join(dataRoot, "uploads")
	for _, directory := range []string{backupsRoot, filepath.Join(uploadsRoot, "logos"), filepath.Join(uploadsRoot, "backgrounds"), filepath.Join(uploadsRoot, "site")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	databasePath := filepath.Join(dataRoot, "nav.db")
	store, err := storage.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	category, err := store.CreateCategory(context.Background(), navigation.CategoryInput{Name: "Tools", Slug: "tools", Visibility: navigation.VisibilityPrivate, SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateLink(context.Background(), navigation.LinkInput{CategoryID: category.ID, Name: "Example", URL: "https://example.com", IconSource: navigation.IconSourceGenerated, IconValue: "E", SortOrder: 10}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.CreateSession(context.Background(), auth.Session{TokenHash: make([]byte, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	uploadContent := []byte("normalized image content")
	if err := os.WriteFile(filepath.Join(uploadsRoot, "logos", "logo.png"), uploadContent, 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := backup.New(backup.Config{Root: backupsRoot, DatabasePath: databasePath, UploadsPath: uploadsRoot, ApplicationVersion: "test-version", Database: store, AvailableSpace: func(string) (uint64, error) { return math.MaxUint64, nil }})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.Create(context.Background())
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	listed, err := service.List()
	if err != nil || len(listed) != 1 || listed[0].Name != created.Name {
		t.Fatalf("unexpected backup list: %+v err=%v", listed, err)
	}
	archive, err := zip.OpenReader(filepath.Join(backupsRoot, created.Name))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	entries := make(map[string]*zip.File)
	for _, entry := range archive.File {
		entries[entry.Name] = entry
		if strings.HasPrefix(entry.Name, "backups/") || strings.Contains(entry.Name, ".backup-creating-") {
			t.Fatalf("archive contains temporary or recursive data: %s", entry.Name)
		}
	}
	for _, name := range []string{"nav.db", "uploads/", "uploads/logos/logo.png", "manifest.json"} {
		if entries[name] == nil {
			t.Fatalf("backup entry %q is missing", name)
		}
	}
	manifest := readManifest(t, entries["manifest.json"])
	if manifest.ApplicationVersion != "test-version" || manifest.SchemaVersion != storage.LatestSchemaVersion || manifest.Format != backup.ManifestFormat {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	verifyManifestFiles(t, manifest, entries)
	extractedDatabase := filepath.Join(root, "restored.db")
	extractEntry(t, entries["nav.db"], extractedDatabase)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := sql.Open("sqlite", extractedDatabase)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	var sessions, links int
	if err := snapshot.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.QueryRow(`SELECT COUNT(*) FROM links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 || links != 1 {
		t.Fatalf("unexpected snapshot content: sessions=%d links=%d", sessions, links)
	}
	opened, openedInfo, err := service.Open(created.Name)
	if err != nil || openedInfo.Size() != created.Size {
		t.Fatalf("open backup: info=%v err=%v", openedInfo, err)
	}
	opened.Close()
	if err := service.Delete(created.Name); err != nil {
		t.Fatal(err)
	}
	if listed, err := service.List(); err != nil || len(listed) != 0 {
		t.Fatalf("deleted backup remained listed: %+v %v", listed, err)
	}
}

func TestBackupFailuresLeaveNoPartialArchive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	databasePath := filepath.Join(root, "nav.db")
	uploadsPath := filepath.Join(root, "uploads")
	if err := os.WriteFile(databasePath, []byte("database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(uploadsPath, 0o700); err != nil {
		t.Fatal(err)
	}
	failing := &fakeDatabase{err: errors.New("snapshot failed")}
	service, err := backup.New(backup.Config{Root: root, DatabasePath: databasePath, UploadsPath: uploadsPath, Database: failing, AvailableSpace: func(string) (uint64, error) { return math.MaxUint64, nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background()); err == nil {
		t.Fatal("expected snapshot failure")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".backup-creating-") || strings.HasSuffix(entry.Name(), ".zip") {
			t.Fatalf("partial backup remained: %s", entry.Name())
		}
	}
	lowSpace, _ := backup.New(backup.Config{Root: root, DatabasePath: databasePath, UploadsPath: uploadsPath, Database: failing, AvailableSpace: func(string) (uint64, error) { return 0, nil }})
	if _, err := lowSpace.Create(context.Background()); !errors.Is(err, backup.ErrInsufficientSpace) {
		t.Fatalf("expected space error, got %v", err)
	}
	if _, _, err := service.Open(`..\outside.zip`); !errors.Is(err, backup.ErrBackupNotFound) {
		t.Fatalf("unsafe name was accepted: %v", err)
	}
}

type fakeDatabase struct{ err error }

func (database *fakeDatabase) BackupSnapshot(context.Context, string) (int, error) {
	return 0, database.err
}

func readManifest(t *testing.T, entry *zip.File) backup.Manifest {
	t.Helper()
	reader, err := entry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var manifest backup.Manifest
	if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func verifyManifestFiles(t *testing.T, manifest backup.Manifest, entries map[string]*zip.File) {
	t.Helper()
	for _, file := range manifest.Files {
		entry := entries[file.Path]
		if entry == nil {
			t.Fatalf("manifest file is missing: %s", file.Path)
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		size, err := io.Copy(hash, reader)
		reader.Close()
		if err != nil || size != file.Size || hex.EncodeToString(hash.Sum(nil)) != file.SHA256 {
			t.Fatalf("integrity mismatch for %s", file.Path)
		}
	}
}

func extractEntry(t *testing.T, entry *zip.File, destination string) {
	t.Helper()
	reader, err := entry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	file, err := os.Create(destination)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(file, reader); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
