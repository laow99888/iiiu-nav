package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/restore"
)

func TestRestoreEndpointRequiresSessionPasswordAndConfirmation(t *testing.T) {
	t.Parallel()
	service := newHTTPRestoreService(t)
	verifier := &restoreVerifier{}
	handler := New(Config{Auth: &importAuthenticator{}, Restores: service, Reverify: verifier})
	archive := restoreHTTPArchive(t)

	anonymous := restoreRequest(t, archive, "correct password", "RESTORE", false)
	anonymousResponse := httptest.NewRecorder()
	handler.ServeHTTP(anonymousResponse, anonymous)
	if anonymousResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous rejection, got %d", anonymousResponse.Code)
	}

	missingConfirmation := restoreRequest(t, archive, "correct password", "", true)
	confirmationResponse := httptest.NewRecorder()
	handler.ServeHTTP(confirmationResponse, missingConfirmation)
	if confirmationResponse.Code != http.StatusUnprocessableEntity || verifier.calls != 0 {
		t.Fatalf("confirmation should be checked first: %d calls=%d", confirmationResponse.Code, verifier.calls)
	}

	verifier.err = auth.ErrInvalidCredentials
	wrongPassword := restoreRequest(t, archive, "wrong password", "RESTORE", true)
	passwordResponse := httptest.NewRecorder()
	handler.ServeHTTP(passwordResponse, wrongPassword)
	if passwordResponse.Code != http.StatusUnauthorized || !strings.Contains(passwordResponse.Body.String(), "restore_password_invalid") {
		t.Fatalf("unexpected password rejection: %d %s", passwordResponse.Code, passwordResponse.Body.String())
	}
}

func TestRestoreEndpointSwapsDataAndClearsSession(t *testing.T) {
	t.Parallel()
	service := newHTTPRestoreService(t)
	handler := New(Config{Auth: &importAuthenticator{}, Restores: service, Reverify: &restoreVerifier{}})
	request := restoreRequest(t, restoreHTTPArchive(t), "correct password", "RESTORE", true)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "preRestoreBackup") {
		t.Fatalf("unexpected restore response: %d %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != SessionCookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("restore did not clear current session: %+v", cookies)
	}
}

func TestRestoreEndpointCompletesWhenClientDisconnects(t *testing.T) {
	t.Parallel()
	service := newHTTPRestoreService(t)
	handler := New(Config{Auth: &importAuthenticator{}, Restores: service, Reverify: &restoreVerifier{}})
	request := restoreRequest(t, restoreHTTPArchive(t), "correct password", "RESTORE", true)
	canceled, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(canceled)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "preRestoreBackup") {
		t.Fatalf("expected restore to complete despite canceled request context: %d %s", response.Code, response.Body.String())
	}
}

func newHTTPRestoreService(t *testing.T) *restore.Service {
	t.Helper()
	root := t.TempDir()
	database := filepath.Join(root, "nav.db")
	uploads := filepath.Join(root, "uploads")
	if err := os.WriteFile(database, []byte("active"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(uploads, 0o700); err != nil {
		t.Fatal(err)
	}
	service, err := restore.New(restore.Config{
		DataRoot: root, DatabasePath: database, UploadsPath: uploads, CurrentSchema: 1,
		Database: &restoreLifecycle{}, Backups: &restoreBackupCreator{},
		PrepareDatabase: func(context.Context, string, int) (int, error) { return 1, nil },
		AvailableSpace:  func(string) (uint64, error) { return math.MaxUint64, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func restoreHTTPArchive(t *testing.T) []byte {
	t.Helper()
	database := []byte("restored")
	hash := sha256.Sum256(database)
	manifest := backup.Manifest{
		Format: backup.ManifestFormat, Version: backup.ManifestVersion, ApplicationVersion: "test",
		SchemaVersion: 1, CreatedAt: time.Now().UTC(),
		Files: []backup.FileIntegrity{{Path: "nav.db", Size: int64(len(database)), SHA256: hex.EncodeToString(hash[:])}},
	}
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	uploads, _ := archive.Create("uploads/")
	_, _ = uploads.Write(nil)
	databaseEntry, _ := archive.Create("nav.db")
	_, _ = databaseEntry.Write(database)
	manifestEntry, _ := archive.Create("manifest.json")
	_ = json.NewEncoder(manifestEntry).Encode(manifest)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func restoreRequest(t *testing.T, archive []byte, password, confirmation string, authenticated bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("backup", "backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write(archive)
	_ = writer.WriteField("password", password)
	_ = writer.WriteField("confirmation", confirmation)
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/restore", &body)
	request.Host = "nav.test"
	request.Header.Set("Origin", "http://nav.test")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if authenticated {
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	}
	return request
}

type restoreVerifier struct {
	err   error
	calls int
}

func (verifier *restoreVerifier) VerifyPassword(context.Context, string) error {
	verifier.calls++
	return verifier.err
}

type restoreLifecycle struct{}

func (*restoreLifecycle) Close() error { return nil }

// Reopen reports a canceled context like the real SQLite reopen would.
func (*restoreLifecycle) Reopen(ctx context.Context) error { return ctx.Err() }

type restoreBackupCreator struct{}

func (*restoreBackupCreator) Create(context.Context) (backup.Info, error) {
	return backup.Info{Name: "pre-restore.zip"}, nil
}
