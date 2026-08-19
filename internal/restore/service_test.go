package restore_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/navigation"
	"iiiu-nav/internal/restore"
	storage "iiiu-nav/internal/storage/sqlite"
)

func TestRestoreReplacesDataCreatesPreBackupAndInvalidatesSessions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sourceRoot, sourceStore, sourceLayout := restoreTestData(t, "source")
	defer sourceStore.Close()
	if _, err := sourceStore.UpsertAdmin(ctx, "restored-password-hash"); err != nil {
		t.Fatal(err)
	}
	restoredCategory, err := sourceStore.CreateCategory(ctx, navigation.CategoryInput{Name: "Restored", Slug: "restored", Visibility: navigation.VisibilityPrivate, SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sourceStore.CreateLink(ctx, navigation.LinkInput{CategoryID: restoredCategory.ID, Name: "Restored link", URL: "https://restored.example", IconSource: navigation.IconSourceGenerated, IconValue: "R", SortOrder: 10}); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(sourceLayout.uploads, "logos", "restored.png"), "restored-upload")
	sourceBackups := newBackupService(t, sourceLayout, sourceStore)
	sourceInfo, err := sourceBackups.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sourceArchive, err := os.Open(filepath.Join(sourceLayout.backups, sourceInfo.Name))
	if err != nil {
		t.Fatal(err)
	}
	defer sourceArchive.Close()

	_, activeStore, activeLayout := restoreTestData(t, "active")
	defer activeStore.Close()
	if _, err := activeStore.UpsertAdmin(ctx, "active-password-hash"); err != nil {
		t.Fatal(err)
	}
	activeCategory, err := activeStore.CreateCategory(ctx, navigation.CategoryInput{Name: "Active", Slug: "active", Visibility: navigation.VisibilityPublic, SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := activeStore.CreateLink(ctx, navigation.LinkInput{CategoryID: activeCategory.ID, Name: "Active link", URL: "https://active.example", IconSource: navigation.IconSourceGenerated, IconValue: "A", SortOrder: 10}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := activeStore.CreateSession(ctx, auth.Session{TokenHash: make([]byte, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(activeLayout.uploads, "logos", "active.png"), "active-upload")
	activeBackups := newBackupService(t, activeLayout, activeStore)
	restorer, err := restore.New(restore.Config{
		DataRoot: activeLayout.root, DatabasePath: activeLayout.database, UploadsPath: activeLayout.uploads,
		CurrentSchema: storage.LatestSchemaVersion, Database: activeStore, Backups: activeBackups,
		PrepareDatabase: storage.PrepareRestoreCandidate, IncompatibleDBError: storage.ErrIncompatibleSchema,
		AvailableSpace: func(string) (uint64, error) { return math.MaxUint64, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := restorer.Restore(ctx, sourceArchive, sourceInfo.Size)
	if err != nil {
		t.Fatalf("restore backup: %v", err)
	}
	if result.PreRestoreBackup.Name == "" {
		t.Fatal("pre-restore backup was not created")
	}
	groups, err := activeStore.Navigation(ctx, true)
	if err != nil || len(groups) != 1 || groups[0].Category.Name != "Restored" || groups[0].Links[0].Name != "Restored link" {
		t.Fatalf("active database was not replaced: %+v err=%v", groups, err)
	}
	if _, err := os.Stat(filepath.Join(activeLayout.uploads, "logos", "restored.png")); err != nil {
		t.Fatalf("restored upload is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(activeLayout.uploads, "logos", "active.png")); !os.IsNotExist(err) {
		t.Fatalf("old upload remained active: %v", err)
	}
	if _, found, err := activeStore.Session(ctx, make([]byte, 32)); err != nil || found {
		t.Fatalf("restored sessions were not invalidated: found=%v err=%v", found, err)
	}
	if _, err := os.Stat(sourceRoot); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsMaliciousCorruptIncompatibleAndOversizeArchives(t *testing.T) {
	t.Parallel()
	service := validationOnlyService(t)
	tests := []struct {
		name string
		data []byte
		want error
	}{
		{name: "traversal", data: makeArchive(t, map[string][]byte{"nav.db": []byte("db"), "uploads/../evil": []byte("bad")}, validManifest(map[string][]byte{"nav.db": []byte("db")})), want: restore.ErrArchiveInvalid},
		{name: "checksum", data: makeArchive(t, map[string][]byte{"nav.db": []byte("changed")}, validManifest(map[string][]byte{"nav.db": []byte("expected")})), want: restore.ErrArchiveInvalid},
		{name: "schema", data: makeArchive(t, map[string][]byte{"nav.db": []byte("db")}, manifestWithSchema(map[string][]byte{"nav.db": []byte("db")}, 99)), want: restore.ErrArchiveIncompatible},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Restore(context.Background(), bytes.NewReader(test.data), int64(len(test.data)))
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
	if _, err := service.Restore(context.Background(), bytes.NewReader(nil), restore.MaxCompressedBytes+1); !errors.Is(err, restore.ErrArchiveTooLarge) {
		t.Fatalf("expected compressed size rejection, got %v", err)
	}
}

func TestRestoreRollsBackWhenReopenFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	databasePath := filepath.Join(root, "nav.db")
	uploadsPath := filepath.Join(root, "uploads")
	writeTestFile(t, databasePath, "active-database")
	writeTestFile(t, filepath.Join(uploadsPath, "active.txt"), "active-upload")
	database := &failingLifecycle{failFirstReopen: true}
	service, err := restore.New(restore.Config{
		DataRoot: root, DatabasePath: databasePath, UploadsPath: uploadsPath, CurrentSchema: 1,
		Database: database, Backups: &fakeBackupCreator{},
		PrepareDatabase: func(context.Context, string, int) (int, error) { return 1, nil },
		AvailableSpace:  func(string) (uint64, error) { return math.MaxUint64, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"nav.db": []byte("restored-database"), "uploads/logos/restored.png": []byte("restored-upload")}
	archive := makeArchive(t, files, validManifest(files))
	_, err = service.Restore(context.Background(), bytes.NewReader(archive), int64(len(archive)))
	if err == nil || !strings.Contains(err.Error(), "reopen restored database") {
		t.Fatalf("expected reopen failure, got %v", err)
	}
	if content, _ := os.ReadFile(databasePath); string(content) != "active-database" {
		t.Fatalf("database was not rolled back: %q", content)
	}
	if content, _ := os.ReadFile(filepath.Join(uploadsPath, "active.txt")); string(content) != "active-upload" {
		t.Fatalf("uploads were not rolled back: %q", content)
	}
	if database.reopenCalls != 2 {
		t.Fatalf("expected failed reopen plus rollback reopen, got %d", database.reopenCalls)
	}
}

func TestRestoreKeepsActiveDataWhenInitialMoveFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	databasePath := filepath.Join(root, "nav.db")
	uploadsPath := filepath.Join(root, "uploads")
	writeTestFile(t, databasePath, "active-database")
	writeTestFile(t, filepath.Join(uploadsPath, "active.txt"), "active-upload")
	database := &failingLifecycle{}
	service, err := restore.New(restore.Config{
		DataRoot: root, DatabasePath: databasePath, UploadsPath: uploadsPath, CurrentSchema: 1,
		Database: database, Backups: &fakeBackupCreator{},
		PrepareDatabase: func(context.Context, string, int) (int, error) { return 1, nil },
		AvailableSpace:  func(string) (uint64, error) { return math.MaxUint64, nil },
		Rename: func(source, destination string) error {
			if source == databasePath {
				return errors.New("initial move failed")
			}
			return os.Rename(source, destination)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"nav.db": []byte("restored-database")}
	archive := makeArchive(t, files, validManifest(files))
	_, err = service.Restore(context.Background(), bytes.NewReader(archive), int64(len(archive)))
	if err == nil || !strings.Contains(err.Error(), "stage active database") {
		t.Fatalf("expected initial move failure, got %v", err)
	}
	if content, _ := os.ReadFile(databasePath); string(content) != "active-database" {
		t.Fatalf("active database was displaced: %q", content)
	}
	if content, _ := os.ReadFile(filepath.Join(uploadsPath, "active.txt")); string(content) != "active-upload" {
		t.Fatalf("active uploads were displaced: %q", content)
	}
	if database.reopenCalls != 1 {
		t.Fatalf("active database was not reopened: %d", database.reopenCalls)
	}
}

type testLayout struct{ root, database, uploads, backups string }

func restoreTestData(t *testing.T, name string) (string, *storage.Store, testLayout) {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	layout := testLayout{root: root, database: filepath.Join(root, "nav.db"), uploads: filepath.Join(root, "uploads"), backups: filepath.Join(root, "backups")}
	for _, directory := range []string{filepath.Join(layout.uploads, "logos"), filepath.Join(layout.uploads, "backgrounds"), filepath.Join(layout.uploads, "site"), layout.backups} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	store, err := storage.Open(context.Background(), layout.database)
	if err != nil {
		t.Fatal(err)
	}
	return root, store, layout
}

func newBackupService(t *testing.T, layout testLayout, store *storage.Store) *backup.Service {
	t.Helper()
	service, err := backup.New(backup.Config{Root: layout.backups, DatabasePath: layout.database, UploadsPath: layout.uploads, ApplicationVersion: "test", Database: store, AvailableSpace: func(string) (uint64, error) { return math.MaxUint64, nil }})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func validationOnlyService(t *testing.T) *restore.Service {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "nav.db"), "active")
	if err := os.MkdirAll(filepath.Join(root, "uploads"), 0o700); err != nil {
		t.Fatal(err)
	}
	service, err := restore.New(restore.Config{
		DataRoot: root, DatabasePath: filepath.Join(root, "nav.db"), UploadsPath: filepath.Join(root, "uploads"), CurrentSchema: 1,
		Database: &failingLifecycle{}, Backups: &fakeBackupCreator{},
		PrepareDatabase: func(context.Context, string, int) (int, error) { return 1, nil },
		AvailableSpace:  func(string) (uint64, error) { return math.MaxUint64, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func makeArchive(t *testing.T, files map[string][]byte, manifest backup.Manifest) []byte {
	t.Helper()
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	_, _ = archive.Create("uploads/")
	for name, content := range files {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	entry, _ := archive.Create("manifest.json")
	if err := json.NewEncoder(entry).Encode(manifest); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func validManifest(files map[string][]byte) backup.Manifest { return manifestWithSchema(files, 1) }

func manifestWithSchema(files map[string][]byte, schema int) backup.Manifest {
	manifest := backup.Manifest{Format: backup.ManifestFormat, Version: backup.ManifestVersion, ApplicationVersion: "test", SchemaVersion: schema, CreatedAt: time.Now().UTC(), Files: make([]backup.FileIntegrity, 0, len(files))}
	for name, content := range files {
		hash := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, backup.FileIntegrity{Path: name, Size: int64(len(content)), SHA256: hex.EncodeToString(hash[:])})
	}
	return manifest
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

type failingLifecycle struct {
	failFirstReopen bool
	reopenCalls     int
}

func (*failingLifecycle) Close() error { return nil }
func (database *failingLifecycle) Reopen(context.Context) error {
	database.reopenCalls++
	if database.failFirstReopen && database.reopenCalls == 1 {
		return errors.New("reopen failed")
	}
	return nil
}

type fakeBackupCreator struct{}

func (*fakeBackupCreator) Create(context.Context) (backup.Info, error) {
	return backup.Info{Name: "pre-restore.zip"}, nil
}
