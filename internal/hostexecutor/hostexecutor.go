// Package hostexecutor implements the restricted update executor that runs
// beside the Docker daemon. It exposes exactly one operation: move the fixed
// iiiu-nav Compose service to a release whose image digest is verified
// against the official GitHub release manifest. Other projects, services,
// images, commands, and host paths are unreachable by construction: every
// command is assembled from fixed configuration plus validated tokens.
package hostexecutor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Executor states.
const (
	StateIdle      = "idle"
	StateRunning   = "running"
	StateSucceeded = "succeeded"
	StateFailed    = "failed"
)

// Phases reported while an update is in flight.
const (
	PhaseQueued        = "queued"
	PhaseVerifyRelease = "verifying_release"
	PhasePulling       = "pulling"
	PhaseRestarting    = "restarting"
	PhaseVerifyHealth  = "verifying_health"
	PhaseRecovering    = "recovering"
)

const (
	ManifestName     = "release-manifest.json"
	OverrideName     = ".executor-image.yaml"
	maxReleaseBytes  = 1 << 20
	maxManifestBytes = 64 << 10
	manifestSchema   = 1
)

var (
	ErrUpdateRunning  = errors.New("an update is already in progress")
	ErrInvalidVersion = errors.New("invalid target version")

	versionPattern = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)
	digestPattern  = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

// Status reports the executor state machine position. States are idle,
// running, succeeded, and failed; phases describe the in-flight step so the
// browser can render progress.
type Status struct {
	State         string    `json:"state"`
	Phase         string    `json:"phase"`
	TargetVersion string    `json:"targetVersion,omitempty"`
	Message       string    `json:"message,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// StartRequest asks the executor to move the service to one exact stable
// version. BackupName is the pre-update backup relative to the data
// directory; it enables data recovery when a failed version migrated the
// schema before the old binary can start again.
type StartRequest struct {
	Version       string
	CurrentSchema int
	BackupName    string
}

type Config struct {
	Repository   string   // fixed image repository, e.g. ghcr.io/laow99888/iiiu-nav
	ProjectName  string   // Compose project name; empty derives it from the compose file
	ProjectDir   string   // Compose project directory
	ComposeFiles []string // base file first, executor override file last
	OverrideFile string   // executor-owned override file, relative to ProjectDir
	Service      string   // Compose service name ("app")
	ReleasesURL  string   // GitHub releases prefix, e.g. https://api.github.com/repos/OWNER/REPO/releases/tags/
	HTTPClient   *http.Client
	Now          func() time.Time
}

// Runner executes fixed allow-listed commands. The production adapter shells
// out to docker; tests inject a fake.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type Service struct {
	config Config
	runner Runner
	clock  func() time.Time

	mu     sync.Mutex
	status Status
}

func New(config Config, runner Runner) *Service {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if config.OverrideFile == "" {
		config.OverrideFile = OverrideName
	}
	clock := config.Now
	if clock == nil {
		clock = time.Now
	}
	return &Service{
		config: config,
		runner: runner,
		clock:  clock,
		status: Status{State: StateIdle, Phase: PhaseQueued, UpdatedAt: clock()},
	}
}

func (service *Service) Status() Status {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.status
}

// Start validates the request and performs the update asynchronously; callers
// poll Status for progress. Only one update may run at a time.
func (service *Service) Start(request StartRequest) error {
	if !versionPattern.MatchString(request.Version) {
		return ErrInvalidVersion
	}
	service.mu.Lock()
	if service.status.State == StateRunning {
		service.mu.Unlock()
		return ErrUpdateRunning
	}
	service.setStatusLocked(Status{State: StateRunning, Phase: PhaseQueued, TargetVersion: request.Version, UpdatedAt: service.clock()})
	go service.perform(request)
	service.mu.Unlock()
	return nil
}

func (service *Service) perform(request StartRequest) {
	ctx := context.Background()
	image, err := service.resolveRelease(ctx, request.Version)
	if err != nil {
		service.finish(request, StateFailed, "release verification failed: "+err.Error())
		return
	}
	if err := service.run(ctx, PhasePulling, "docker", "pull", image); err != nil {
		service.finish(request, StateFailed, "image pull failed: "+err.Error())
		return
	}
	if err := service.swap(ctx, image); err != nil {
		service.recover(ctx, request, err.Error())
		return
	}
	service.finish(request, StateSucceeded, "")
}

// swap rewrites the executor-owned override file to the exact digest reference
// and recreates the service; up --wait returns only once the health check
// passes, so a successful swap implies a healthy application.
func (service *Service) swap(ctx context.Context, image string) error {
	service.setPhase(PhaseRestarting, "")
	if err := service.writeOverride(image); err != nil {
		return err
	}
	service.setPhase(PhaseVerifyHealth, "")
	return service.upWait(ctx)
}

// recover returns the service to the previous image. When even the previous
// image stays unhealthy, the failed version may have advanced the database
// schema past what the old binary accepts; restoring the pre-update backup
// then makes the previous image bootable again.
func (service *Service) recover(ctx context.Context, request StartRequest, cause string) {
	message := "update failed: " + cause
	service.setPhase(PhaseRecovering, message)
	previous := service.previousImageRef(request)
	if err := service.writeOverride(previous); err != nil {
		service.finish(request, StateFailed, message+"; rollback failed: "+err.Error())
		return
	}
	if err := service.upWait(ctx); err != nil {
		// The old binary may be refusing a database the failed version
		// already migrated; restoring the pre-update backup makes it bootable
		// again.
		if request.BackupName == "" {
			service.finish(request, StateFailed, message+"; previous image is unhealthy and no pre-update backup was provided")
			return
		}
		restorePath := filepath.ToSlash(filepath.Join("/data/backups", request.BackupName))
		// The restore swaps database files under /data; no live process may
		// hold them open while a one-off restore container works on the
		// volume.
		if _, err := service.runner.Run(ctx, "docker", service.composeArgs("stop", service.config.Service)...); err != nil {
			service.finish(request, StateFailed, message+"; could not stop the service before backup restore: "+err.Error())
			return
		}
		if _, err := service.runner.Run(ctx, "docker", service.composeArgs("run", "--rm", service.config.Service, "restore", restorePath)...); err != nil {
			service.finish(request, StateFailed, message+"; backup restore failed: "+err.Error())
			return
		}
		if err := service.upWait(ctx); err != nil {
			service.finish(request, StateFailed, message+"; still unhealthy after restoring the pre-update backup")
			return
		}
		service.finish(request, StateFailed, message+"; previous image is healthy again after restoring the pre-update backup")
		return
	}
	service.finish(request, StateFailed, message+"; previous image is healthy again")
}

// resolveRelease independently verifies release identity against GitHub: the
// application can only nominate a version; the digest comes from the official
// manifest or not at all.
func (service *Service) resolveRelease(ctx context.Context, version string) (string, error) {
	release, err := fetchJSON(ctx, service.config.HTTPClient, service.config.ReleasesURL+version, maxReleaseBytes)
	if err != nil {
		return "", fmt.Errorf("release lookup: %w", err)
	}
	manifestURL, err := manifestAssetURL(release, version)
	if err != nil {
		return "", err
	}
	payload, err := fetchJSON(ctx, service.config.HTTPClient, manifestURL, maxManifestBytes)
	if err != nil {
		return "", fmt.Errorf("manifest download: %w", err)
	}
	var manifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		Version       string `json:"version"`
		Repository    string `json:"repository"`
		Digest        string `json:"digest"`
		Image         string `json:"image"`
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("manifest shape: %w", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", fmt.Errorf("manifest shape: %w", err)
	}
	switch {
	case manifest.SchemaVersion != manifestSchema:
		return "", fmt.Errorf("unsupported manifest schema %d", manifest.SchemaVersion)
	case manifest.Version != version:
		return "", fmt.Errorf("manifest version %q does not match target", manifest.Version)
	case manifest.Repository != service.config.Repository:
		return "", fmt.Errorf("manifest repository %q is not the fixed repository", manifest.Repository)
	case !digestPattern.MatchString(manifest.Digest):
		return "", fmt.Errorf("manifest digest is not a valid sha256 reference")
	case manifest.Image != manifest.Repository+"@"+manifest.Digest:
		return "", fmt.Errorf("manifest image reference is inconsistent")
	}
	return manifest.Image, nil
}

func fetchJSON(ctx context.Context, client *http.Client, url string, limit int64) (any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "iiiu-nav-update-executor")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, errors.New("response too large")
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func manifestAssetURL(release any, version string) (string, error) {
	object, ok := release.(map[string]any)
	if !ok {
		return "", errors.New("release payload is not an object")
	}
	if tag, _ := object["tag_name"].(string); tag != version {
		return "", fmt.Errorf("release tag %q does not match target", tag)
	}
	if draft, _ := object["draft"].(bool); draft {
		return "", errors.New("target release is a draft")
	}
	if prerelease, _ := object["prerelease"].(bool); prerelease {
		return "", errors.New("target release is a prerelease")
	}
	assets, _ := object["assets"].([]any)
	for _, asset := range assets {
		entry, ok := asset.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		url, _ := entry["browser_download_url"].(string)
		if name == ManifestName && url != "" {
			return url, nil
		}
	}
	return "", fmt.Errorf("release has no %s asset", ManifestName)
}

// previousImageRef captures the rollback target: the reference the running
// container was created with, falling back to the documented stable tag.
func (service *Service) previousImageRef(request StartRequest) string {
	project := filepath.Base(service.config.ProjectDir)
	output, err := service.runner.Run(context.Background(), "docker",
		"inspect", project+"-"+service.config.Service+"-1", "--format", "{{.Config.Image}}")
	if err == nil {
		if reference := strings.TrimSpace(output); reference != "" {
			return reference
		}
	}
	return service.config.Repository + ":stable"
}

func (service *Service) upWait(ctx context.Context) error {
	return service.compose(ctx, "up", "-d", "--wait", service.config.Service)
}

func (service *Service) compose(ctx context.Context, args ...string) error {
	_, err := service.runner.Run(ctx, "docker", service.composeArgs(args...)...)
	return err
}

func (service *Service) composeArgs(args ...string) []string {
	command := []string{"compose"}
	if service.config.ProjectName != "" {
		command = append(command, "--project-name", service.config.ProjectName)
	}
	command = append(command, "--project-directory", service.config.ProjectDir)
	for _, file := range service.config.ComposeFiles {
		command = append(command, "--file", file)
	}
	return append(command, args...)
}

func (service *Service) run(ctx context.Context, phase, name string, args ...string) error {
	if phase != "" {
		service.setPhase(phase, "")
	}
	_, err := service.runner.Run(ctx, name, args...)
	return err
}

// writeOverride owns exactly one file inside the project directory: the
// executor-rendered image override. Nothing else in the project is touched.
func (service *Service) writeOverride(image string) error {
	path := filepath.Join(service.config.ProjectDir, service.config.OverrideFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o770); err != nil {
		return err
	}
	content := "services:\n  " + service.config.Service + ":\n    image: " + image + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return err
	}
	return nil
}

func (service *Service) setPhase(phase, message string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.status.Phase = phase
	if message != "" {
		service.status.Message = message
	}
	service.status.UpdatedAt = service.clock()
}

func (service *Service) setStatusLocked(status Status) {
	service.status = status
}

func (service *Service) finish(request StartRequest, state, message string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.status = Status{
		State:         state,
		Phase:         PhaseQueued,
		TargetVersion: request.Version,
		Message:       message,
		UpdatedAt:     service.clock(),
	}
}
