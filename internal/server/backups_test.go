package server

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"iiiu-nav/internal/backup"
	storage "iiiu-nav/internal/storage/sqlite"
)

func TestAuthenticatedBackupLifecycle(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	uploads := filepath.Join(root, "uploads")
	backups := filepath.Join(root, "backups")
	if err := os.MkdirAll(filepath.Join(uploads, "logos"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(backups, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "logos", "test.png"), []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "nav.db")
	store, err := storage.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service, err := backup.New(backup.Config{Root: backups, DatabasePath: databasePath, UploadsPath: uploads, ApplicationVersion: "test", Database: store, AvailableSpace: func(string) (uint64, error) { return math.MaxUint64, nil }})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Config{Auth: &importAuthenticator{}, Backups: service})

	createRequest := backupRequest(http.MethodPost, "/api/backups")
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create backup: %d %s", createResponse.Code, createResponse.Body.String())
	}
	var created backup.Info
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil || created.Name == "" {
		t.Fatalf("decode created backup: %+v %v", created, err)
	}

	listRequest := backupRequest(http.MethodGet, "/api/backups")
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), created.Name) || listResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("list backups: %d %s", listResponse.Code, listResponse.Body.String())
	}

	downloadRequest := backupRequest(http.MethodGet, "/api/backups/"+created.Name)
	downloadResponse := httptest.NewRecorder()
	handler.ServeHTTP(downloadResponse, downloadRequest)
	if downloadResponse.Code != http.StatusOK || downloadResponse.Header().Get("Content-Type") != "application/zip" || !strings.HasPrefix(downloadResponse.Body.String(), "PK") {
		t.Fatalf("download backup: %d headers=%v", downloadResponse.Code, downloadResponse.Header())
	}

	deleteRequest := backupRequest(http.MethodDelete, "/api/backups/"+created.Name)
	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete backup: %d %s", deleteResponse.Code, deleteResponse.Body.String())
	}
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, backupRequest(http.MethodGet, "/api/backups/"+created.Name))
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected deleted backup to be missing, got %d", missingResponse.Code)
	}
}

func TestBackupEndpointsRejectAnonymousRequests(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	databasePath := filepath.Join(root, "nav.db")
	uploads := filepath.Join(root, "uploads")
	if err := os.WriteFile(databasePath, []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(uploads, 0o700); err != nil {
		t.Fatal(err)
	}
	service, _ := backup.New(backup.Config{Root: root, DatabasePath: databasePath, UploadsPath: uploads, Database: &failingBackupDatabase{}})
	handler := New(Config{Auth: &importAuthenticator{}, Backups: service})
	request := httptest.NewRequest(http.MethodGet, "/api/backups", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous rejection, got %d", response.Code)
	}
}

func backupRequest(method, target string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.Host = "nav.test"
	request.Header.Set("Origin", "http://nav.test")
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	return request
}

type failingBackupDatabase struct{}

func (*failingBackupDatabase) BackupSnapshot(context.Context, string) (int, error) {
	return 0, context.Canceled
}
