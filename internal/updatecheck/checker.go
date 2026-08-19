package updatecheck

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	latestReleaseURL = "https://api.github.com/repos/laow99888/iiiu-nav/releases/latest"
	manifestName     = "release-manifest.json"
	maximumBodyBytes = 64 << 10
	defaultCacheTTL  = 6 * time.Hour
)

type State string

const (
	StateDevelopment     State = "development"
	StateUpToDate        State = "up_to_date"
	StateUpdateAvailable State = "update_available"
	StateUnavailable     State = "unavailable"
	StateRateLimited     State = "rate_limited"
	StateInvalidRelease  State = "invalid_release"
)

type Result struct {
	State          State      `json:"state"`
	CurrentVersion string     `json:"currentVersion"`
	LatestVersion  string     `json:"latestVersion,omitempty"`
	ReleaseURL     string     `json:"releaseUrl,omitempty"`
	ManifestURL    string     `json:"manifestUrl,omitempty"`
	PublishedAt    *time.Time `json:"publishedAt,omitempty"`
	CheckedAt      *time.Time `json:"checkedAt,omitempty"`
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Config struct {
	CurrentVersion string
	Client         HTTPDoer
	Now            func() time.Time
	CacheTTL       time.Duration
}

type Checker struct {
	currentVersion string
	client         HTTPDoer
	now            func() time.Time
	cacheTTL       time.Duration

	mu     sync.Mutex
	cached *Result
}

func New(config Config) *Checker {
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	return &Checker{
		currentVersion: strings.TrimSpace(config.CurrentVersion),
		client:         client,
		now:            now,
		cacheTTL:       cacheTTL,
	}
}

func (checker *Checker) Check(ctx context.Context, force bool) Result {
	if _, valid := parseStableVersion(checker.currentVersion); !valid {
		return Result{State: StateDevelopment, CurrentVersion: checker.currentVersion}
	}

	checker.mu.Lock()
	defer checker.mu.Unlock()

	now := checker.now().UTC()
	if !force && checker.cached != nil && checker.cached.CheckedAt != nil && now.Sub(*checker.cached.CheckedAt) < checker.cacheTTL {
		return *checker.cached
	}

	result := checker.fetch(ctx, now)
	if result.State == StateUpToDate || result.State == StateUpdateAvailable {
		checker.cached = &result
	}
	return result
}

func (checker *Checker) fetch(ctx context.Context, checkedAt time.Time) Result {
	result := Result{State: StateUnavailable, CurrentVersion: checker.currentVersion, CheckedAt: &checkedAt}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return result
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "iiiu-nav/1 release checker")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	response, err := checker.client.Do(request)
	if err != nil {
		return result
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
		result.State = StateRateLimited
		return result
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maximumBodyBytes+1))
	if err != nil || len(body) > maximumBodyBytes {
		return result
	}
	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		result.State = StateInvalidRelease
		return result
	}
	latest, valid := parseStableVersion(release.TagName)
	current, currentValid := parseStableVersion(checker.currentVersion)
	manifestURL := release.manifestURL()
	if !valid || !currentValid || release.Draft || release.Prerelease || release.PublishedAt.IsZero() || !validReleaseURL(release.HTMLURL, release.TagName) || manifestURL == "" {
		result.State = StateInvalidRelease
		return result
	}

	publishedAt := release.PublishedAt.UTC()
	result.LatestVersion = release.TagName
	result.ReleaseURL = release.HTMLURL
	result.ManifestURL = manifestURL
	result.PublishedAt = &publishedAt
	if compareVersions(latest, current) > 0 {
		result.State = StateUpdateAvailable
	} else {
		result.State = StateUpToDate
	}
	return result
}

type githubRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func (release githubRelease) manifestURL() string {
	for _, asset := range release.Assets {
		if asset.Name == manifestName && validManifestURL(asset.BrowserDownloadURL, release.TagName) {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

func validReleaseURL(rawURL, version string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "github.com" && parsed.Path == "/laow99888/iiiu-nav/releases/tag/"+version
}

func validManifestURL(rawURL, version string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "github.com" && parsed.Path == "/laow99888/iiiu-nav/releases/download/"+version+"/"+manifestName
}

func parseStableVersion(value string) ([3]uint64, bool) {
	var version [3]uint64
	if !strings.HasPrefix(value, "v") {
		return version, false
	}
	parts := strings.Split(strings.TrimPrefix(value, "v"), ".")
	if len(parts) != len(version) {
		return version, false
	}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return version, false
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return version, false
		}
		version[index] = parsed
	}
	return version, true
}

func compareVersions(left, right [3]uint64) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}
