package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/updatecheck"
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
