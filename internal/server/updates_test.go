package server

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/backup"
	storage "iiiu-nav/internal/storage/sqlite"
	"iiiu-nav/internal/updatecheck"
	"iiiu-nav/internal/updateexecutor"
)

func TestUpdateRoutesRequireAdministratorAndDistinguishManualRefresh(t *testing.T) {
	t.Parallel()
	authenticator := &fakeAuthenticator{}
	checker := &fakeUpdateChecker{result: updatecheck.Result{
		State: updatecheck.StateUpdateAvailable, CurrentVersion: "v1.0.0", LatestVersion: "v1.1.0",
	}}
	handler := New(Config{Auth: authenticator, Updates: checker, Version: "v1.0.0"})

	request := httptest.NewRequest(http.MethodGet, "/api/updates", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "valid"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"state":"update_available"`) || !strings.Contains(response.Body.String(), `"automaticUpdate":false`) || checker.force {
		t.Fatalf("unexpected cached check: status=%d body=%s force=%t", response.Code, response.Body.String(), checker.force)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("update status must not be browser cached")
	}

	manual := httptest.NewRequest(http.MethodPost, "/api/updates/check", nil)
	manual.Host = "nav.test"
	manual.Header.Set("Origin", "http://nav.test")
	manual.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "valid"})
	manualResponse := httptest.NewRecorder()
	handler.ServeHTTP(manualResponse, manual)
	if manualResponse.Code != http.StatusOK || !checker.force {
		t.Fatalf("manual check failed: status=%d body=%s force=%t", manualResponse.Code, manualResponse.Body.String(), checker.force)
	}

	authenticator.authenticateErr = auth.ErrUnauthenticated
	anonymous := httptest.NewRecorder()
	handler.ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/api/updates", nil))
	if anonymous.Code != http.StatusUnauthorized || checker.calls != 2 {
		t.Fatalf("anonymous request reached checker: status=%d calls=%d", anonymous.Code, checker.calls)
	}
}

type fakeUpdateChecker struct {
	calls  int
	force  bool
	result updatecheck.Result
}

func (checker *fakeUpdateChecker) Check(_ context.Context, force bool) updatecheck.Result {
	checker.calls++
	checker.force = force
	return checker.result
}

type fakeUpdateExecutor struct {
	available bool
	started   []updateexecutor.StartRequest
	startErr  error
	status    updateexecutor.Status
	statusErr error
}

func (executor *fakeUpdateExecutor) Available() bool { return executor.available }

func (executor *fakeUpdateExecutor) Start(_ context.Context, request updateexecutor.StartRequest) error {
	if executor.startErr != nil {
		return executor.startErr
	}
	executor.started = append(executor.started, request)
	return nil
}

func (executor *fakeUpdateExecutor) Status(context.Context) (updateexecutor.Status, error) {
	return executor.status, executor.statusErr
}

func newUpdateInstallRequest(t *testing.T, password, version string) *http.Request {
	t.Helper()
	body := `{"password":` + strconv.Quote(password) + `,"version":` + strconv.Quote(version) + `}`
	request := httptest.NewRequest(http.MethodPost, "/api/updates/install", strings.NewReader(body))
	request.Host = "nav.test"
	request.Header.Set("Origin", "http://nav.test")
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	return request
}

func TestOneClickUpdateInstallFlow(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	uploads := filepath.Join(root, "uploads")
	backupsDir := filepath.Join(root, "backups")
	if err := os.MkdirAll(filepath.Join(uploads, "logos"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(backupsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "nav.db")
	store, err := storage.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	backupService, err := backup.New(backup.Config{Root: backupsDir, DatabasePath: databasePath, UploadsPath: uploads, ApplicationVersion: "test", Database: store, AvailableSpace: func(string) (uint64, error) { return math.MaxUint64, nil }})
	if err != nil {
		t.Fatal(err)
	}
	verifier := &restoreVerifier{}
	executor := &fakeUpdateExecutor{available: true}
	checker := &fakeUpdateChecker{result: updatecheck.Result{
		State: updatecheck.StateUpdateAvailable, CurrentVersion: "v1.0.0", LatestVersion: "v1.1.0",
	}}
	handler := New(Config{
		Auth: &importAuthenticator{}, Reverify: verifier, Updates: checker,
		Executor: executor, Backups: backupService, SchemaVersions: store, Version: "v1.0.0",
	})

	// Wrong password stops the flow before anything else happens.
	verifier.err = auth.ErrInvalidCredentials
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, newUpdateInstallRequest(t, "wrong", "v1.1.0"))
	if denied.Code != http.StatusUnauthorized || !strings.Contains(denied.Body.String(), "update_password_invalid") {
		t.Fatalf("wrong password: %d %s", denied.Code, denied.Body.String())
	}
	if len(executor.started) != 0 {
		t.Fatalf("executor must not run on password failure")
	}

	// Only the exact official latest stable version is installable.
	verifier.err = nil
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, newUpdateInstallRequest(t, "correct password", "v9.9.9"))
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "update_not_available") {
		t.Fatalf("arbitrary version: %d %s", stale.Code, stale.Body.String())
	}

	accepted := httptest.NewRecorder()
	handler.ServeHTTP(accepted, newUpdateInstallRequest(t, "correct password", "v1.1.0"))
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("install: %d %s", accepted.Code, accepted.Body.String())
	}
	if len(executor.started) != 1 {
		t.Fatalf("expected one executor start, got %v", executor.started)
	}
	started := executor.started[0]
	if started.Version != "v1.1.0" || started.BackupName == "" || started.CurrentSchema < 1 {
		t.Fatalf("unexpected start request: %+v", started)
	}
	if _, err := os.Stat(filepath.Join(backupsDir, started.BackupName)); err != nil {
		t.Fatalf("pre-update backup must exist: %v", err)
	}

	// The status view now offers one-click updates and proxies progress.
	listing := httptest.NewRecorder()
	listingRequest := httptest.NewRequest(http.MethodGet, "/api/updates", nil)
	listingRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	handler.ServeHTTP(listing, listingRequest)
	if !strings.Contains(listing.Body.String(), `"automaticUpdate":true`) {
		t.Fatalf("automaticUpdate flag missing: %s", listing.Body.String())
	}
	executor.status = updateexecutor.Status{State: "running", Phase: "pulling", TargetVersion: "v1.1.0"}
	progress := httptest.NewRecorder()
	progressRequest := httptest.NewRequest(http.MethodGet, "/api/updates/install/status", nil)
	progressRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	handler.ServeHTTP(progress, progressRequest)
	if progress.Code != http.StatusOK || !strings.Contains(progress.Body.String(), `"phase":"pulling"`) {
		t.Fatalf("status proxy: %d %s", progress.Code, progress.Body.String())
	}

	// A second concurrent update is rejected while one is in flight.
	executor.startErr = updateexecutor.ErrBusy
	busy := httptest.NewRecorder()
	handler.ServeHTTP(busy, newUpdateInstallRequest(t, "correct password", "v1.1.0"))
	if busy.Code != http.StatusConflict || !strings.Contains(busy.Body.String(), "update_already_running") {
		t.Fatalf("busy: %d %s", busy.Code, busy.Body.String())
	}
}

func TestInstallRoutesAbsentWithoutExecutor(t *testing.T) {
	t.Parallel()
	handler := New(Config{Auth: &importAuthenticator{}, Reverify: &restoreVerifier{}, Updates: &fakeUpdateChecker{}, Version: "v1.0.0"})
	install := newUpdateInstallRequest(t, "correct password", "v1.1.0")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, install)
	if response.Code != http.StatusNotFound {
		t.Fatalf("install without executor must 404, got %d", response.Code)
	}
}
