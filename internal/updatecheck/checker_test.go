package updatecheck

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestDevelopmentBuildDoesNotContactGitHub(t *testing.T) {
	t.Parallel()
	checker := New(Config{
		CurrentVersion: "local",
		Client: doerFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("development build contacted GitHub")
			return nil, nil
		}),
	})
	result := checker.Check(context.Background(), false)
	if result.State != StateDevelopment || result.CurrentVersion != "local" || result.CheckedAt != nil {
		t.Fatalf("unexpected development result: %+v", result)
	}
}

func TestStableReleaseComparisonCachingAndManualRefresh(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	calls := 0
	checker := New(Config{
		CurrentVersion: "v1.9.0",
		Now:            func() time.Time { return now },
		Client: doerFunc(func(request *http.Request) (*http.Response, error) {
			calls++
			if request.URL.String() != latestReleaseURL || request.Header.Get("User-Agent") == "" || request.Header.Get("Accept") != "application/vnd.github+json" {
				t.Fatalf("unexpected GitHub request: %s headers=%v", request.URL, request.Header)
			}
			return githubResponse(http.StatusOK, releaseJSON("v1.10.0", false, false)), nil
		}),
	})

	first := checker.Check(context.Background(), false)
	if first.State != StateUpdateAvailable || first.LatestVersion != "v1.10.0" || first.ManifestURL == "" || first.PublishedAt == nil {
		t.Fatalf("unexpected available result: %+v", first)
	}
	now = now.Add(5 * time.Hour)
	second := checker.Check(context.Background(), false)
	if second.CheckedAt == nil || first.CheckedAt == nil || !second.CheckedAt.Equal(*first.CheckedAt) || calls != 1 {
		t.Fatalf("expected cached result, first=%+v second=%+v calls=%d", first, second, calls)
	}
	checker.Check(context.Background(), true)
	if calls != 2 {
		t.Fatalf("manual refresh did not bypass cache: calls=%d", calls)
	}
}

func TestReleaseStatesAreNonDestructiveAndNotFailureCached(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		status     int
		body       string
		want       State
		wantLatest string
	}{
		{name: "up to date", status: http.StatusOK, body: releaseJSON("v1.2.3", false, false), want: StateUpToDate, wantLatest: "v1.2.3"},
		{name: "current ahead", status: http.StatusOK, body: releaseJSON("v1.2.2", false, false), want: StateUpToDate, wantLatest: "v1.2.2"},
		{name: "rate limited", status: http.StatusForbidden, body: `{}`, want: StateRateLimited},
		{name: "unavailable", status: http.StatusBadGateway, body: `{}`, want: StateUnavailable},
		{name: "prerelease", status: http.StatusOK, body: releaseJSON("v1.3.0", false, true), want: StateInvalidRelease},
		{name: "malformed version", status: http.StatusOK, body: releaseJSON("1.3", false, false), want: StateInvalidRelease},
		{name: "missing manifest", status: http.StatusOK, body: strings.Replace(releaseJSON("v1.3.0", false, false), `"assets":[`, `"assets-missing":[`, 1), want: StateInvalidRelease},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			checker := New(Config{
				CurrentVersion: "v1.2.3",
				Client: doerFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return githubResponse(test.status, test.body), nil
				}),
			})
			result := checker.Check(context.Background(), false)
			if result.State != test.want || result.LatestVersion != test.wantLatest {
				t.Fatalf("result=%+v want state=%s latest=%s", result, test.want, test.wantLatest)
			}
			checker.Check(context.Background(), false)
			wantCalls := 2
			if test.want == StateUpToDate {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("calls=%d want %d", calls, wantCalls)
			}
		})
	}
}

func TestStableVersionFormat(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"v0.0.0", "v1.2.3", "v18446744073709551615.2.3"} {
		if _, valid := parseStableVersion(value); !valid {
			t.Fatalf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"", "local", "1.2.3", "v1.2", "v01.2.3", "v1.2.3-beta", "v1.2.3+build", "v1.-2.3"} {
		if _, valid := parseStableVersion(value); valid {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func githubResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func releaseJSON(version string, draft, prerelease bool) string {
	return `{"tag_name":"` + version + `","html_url":"https://github.com/laow99888/iiiu-nav/releases/tag/` + version + `","draft":` + boolString(draft) + `,"prerelease":` + boolString(prerelease) + `,"published_at":"2026-08-20T07:00:00Z","assets":[{"name":"release-manifest.json","browser_download_url":"https://github.com/laow99888/iiiu-nav/releases/download/` + version + `/release-manifest.json"}]}`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
