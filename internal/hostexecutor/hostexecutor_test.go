package hostexecutor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRunner records commands and answers from a scripted error sequence.
// Script entries are consumed one per command; the last entry repeats.
type fakeRunner struct {
	mu       sync.Mutex
	commands []string
	script   []error
}

func (fake *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	command := name + " " + strings.Join(args, " ")
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.commands = append(fake.commands, command)
	var err error
	if len(fake.script) > 0 {
		err = fake.script[0]
		if len(fake.script) > 1 {
			fake.script = fake.script[1:]
		}
	}
	return "", err
}

func (fake *fakeRunner) ran() []string {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return append([]string(nil), fake.commands...)
}

func (fake *fakeRunner) ranCount() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return len(fake.commands)
}

var (
	goodRepo   = "ghcr.io/laow99888/iiiu-nav"
	goodDigest = "sha256:" + strings.Repeat("ab", 32)
)

func manifestJSON(schema int, repo, digest, image string) string {
	return fmt.Sprintf(`{"schemaVersion":%d,"version":"v0.2.0","repository":%q,"digest":%q,"image":%q,"revision":"rev","publishedAt":"2026-09-18T00:00:00Z"}`,
		schema, repo, digest, image)
}

func goodManifest() string {
	return manifestJSON(1, goodRepo, goodDigest, goodRepo+"@"+goodDigest)
}

// newTestService serves the release lookup and the manifest asset from one
// test server and stores the executor override inside a fresh project dir.
func newTestService(t *testing.T, manifest string, runner *fakeRunner) *Service {
	t.Helper()
	var serverURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/tags/v0.2.0", func(writer http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(writer, `{"tag_name":"v0.2.0","draft":false,"prerelease":false,
			"assets":[{"name":"release-manifest.json","browser_download_url":%q},
			{"name":"other.zip","browser_download_url":"ignored"}]}`,
			serverURL+"/manifest/v0.2.0")
	})
	mux.HandleFunc("/manifest/v0.2.0", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(manifest))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	serverURL = server.URL

	projectDir := t.TempDir()
	return New(Config{
		Repository:   goodRepo,
		ProjectDir:   projectDir,
		ComposeFiles: []string{"compose.yaml", OverrideName},
		Service:      "app",
		ReleasesURL:  serverURL + "/releases/tags/",
		Now:          func() time.Time { return time.Unix(0, 0) },
	}, runner)
}

func waitForState(t *testing.T, service *Service, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if service.Status().State == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("state %s not reached, status: %+v", want, service.Status())
}

func overrideContent(t *testing.T, service *Service) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(service.config.ProjectDir, OverrideName))
	if err != nil {
		t.Fatalf("read override: %v", err)
	}
	return string(content)
}

func TestStartRejectsInvalidVersion(t *testing.T) {
	runner := &fakeRunner{}
	service := newTestService(t, goodManifest(), runner)
	for _, version := range []string{"", "stable", "v1.2", "v01.2.3", "v1.2.3.4", "v1.2.3-x"} {
		if err := service.Start(StartRequest{Version: version}); !errors.Is(err, ErrInvalidVersion) {
			t.Fatalf("Start(%q) = %v, want ErrInvalidVersion", version, err)
		}
	}
	if runner.ranCount() != 0 {
		t.Fatalf("unexpected commands: %v", runner.ran())
	}
}

func TestStartRejectsWhileRunning(t *testing.T) {
	service := newTestService(t, goodManifest(), &fakeRunner{})
	service.mu.Lock()
	service.setStatusLocked(Status{State: StateRunning, Phase: PhasePulling, UpdatedAt: time.Unix(0, 0)})
	service.mu.Unlock()
	if err := service.Start(StartRequest{Version: "v0.2.0"}); !errors.Is(err, ErrUpdateRunning) {
		t.Fatalf("Start while running = %v, want ErrUpdateRunning", err)
	}
}

func TestSuccessfulUpdateSequence(t *testing.T) {
	runner := &fakeRunner{}
	service := newTestService(t, goodManifest(), runner)

	if err := service.Start(StartRequest{Version: "v0.2.0", BackupName: "pre.zip"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitForState(t, service, StateSucceeded)

	joined := strings.Join(runner.ran(), "\n")
	if !strings.Contains(joined, "docker pull "+goodRepo+"@"+goodDigest) {
		t.Fatalf("expected pull by exact digest, got:\n%s", joined)
	}
	if !strings.Contains(joined, "up -d --wait app") {
		t.Fatalf("expected health-gated recreate, got:\n%s", joined)
	}
	want := "services:\n  app:\n    image: " + goodRepo + "@" + goodDigest + "\n"
	if got := overrideContent(t, service); got != want {
		t.Fatalf("override content %q, want %q", got, want)
	}
}

func TestFailedHealthRollsBackToPreviousImage(t *testing.T) {
	runner := &fakeRunner{script: []error{nil, errors.New("health check failed"), nil, nil}}
	service := newTestService(t, goodManifest(), runner)

	if err := service.Start(StartRequest{Version: "v0.2.0", BackupName: "pre.zip"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitForState(t, service, StateFailed)

	status := service.Status()
	if !strings.Contains(status.Message, "previous image is healthy again") {
		t.Fatalf("unexpected message %q", status.Message)
	}
	joined := strings.Join(runner.ran(), "\n")
	if !strings.Contains(joined, "-app-1 --format {{.Config.Image}}") {
		t.Fatalf("rollback must capture the previous image:\n%s", joined)
	}
	if !strings.Contains(overrideContent(t, service), goodRepo+":stable") {
		t.Fatalf("override must point back at the previous reference:\n%s", overrideContent(t, service))
	}
}

func TestRollbackRestoresBackupWhenPreviousImageStaysUnhealthy(t *testing.T) {
	runner := &fakeRunner{}
	service := newTestService(t, goodManifest(), runner)
	// pull ok; swap unhealthy; inspect ok; rollback unhealthy; restore ok;
	// retry healthy.
	runner.mu.Lock()
	runner.script = []error{nil, errors.New("unhealthy"), nil, errors.New("old binary refuses schema"), nil, nil}
	runner.mu.Unlock()

	if err := service.Start(StartRequest{Version: "v0.2.0", BackupName: "pre.zip"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitForState(t, service, StateFailed)

	status := service.Status()
	if !strings.Contains(status.Message, "restoring the pre-update backup") {
		t.Fatalf("unexpected message %q", status.Message)
	}
	joined := strings.Join(runner.ran(), "\n")
	if !strings.Contains(joined, "run --rm app restore /data/backups/pre.zip") {
		t.Fatalf("expected backup restore command:\n%s", joined)
	}
}

func TestRollbackWithoutBackupReportsMissingBackup(t *testing.T) {
	runner := &fakeRunner{script: []error{nil, errors.New("unhealthy"), nil, errors.New("old binary refuses schema")}}
	service := newTestService(t, goodManifest(), runner)

	if err := service.Start(StartRequest{Version: "v0.2.0"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitForState(t, service, StateFailed)
	if !strings.Contains(service.Status().Message, "no pre-update backup was provided") {
		t.Fatalf("unexpected message %q", service.Status().Message)
	}
}

func TestReleaseValidationFailures(t *testing.T) {
	cases := map[string]string{
		"draft release":      `{"tag_name":"v0.2.0","draft":true,"prerelease":false,"assets":[]}`,
		"prerelease":         `{"tag_name":"v0.2.0","draft":false,"prerelease":true,"assets":[]}`,
		"tag mismatch":       `{"tag_name":"v9.9.9","draft":false,"prerelease":false,"assets":[]}`,
		"missing manifest":   `{"tag_name":"v0.2.0","draft":false,"prerelease":false,"assets":[]}`,
		"wrong repository":   manifestJSON(1, "ghcr.io/evil/nav", goodDigest, "ghcr.io/evil/nav@"+goodDigest),
		"bad digest":         manifestJSON(1, goodRepo, "sha256:zz", goodRepo+"@sha256:zz"),
		"inconsistent image": manifestJSON(1, goodRepo, goodDigest, goodRepo+":latest"),
		"unknown schema":     manifestJSON(99, goodRepo, goodDigest, goodRepo+"@"+goodDigest),
	}
	for name, manifest := range cases {
		t.Run(name, func(t *testing.T) {
			runner := &fakeRunner{}
			service := newTestService(t, manifest, runner)
			if err := service.Start(StartRequest{Version: "v0.2.0"}); err != nil {
				t.Fatalf("start: %v", err)
			}
			waitForState(t, service, StateFailed)
			status := service.Status()
			if !strings.Contains(status.Message, "release verification failed") {
				t.Fatalf("unexpected message %q", status.Message)
			}
			for _, command := range runner.ran() {
				if strings.Contains(command, "pull") {
					t.Fatalf("must not pull on verification failure: %v", runner.ran())
				}
			}
		})
	}
}
