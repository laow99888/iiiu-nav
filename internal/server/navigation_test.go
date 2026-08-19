package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/navigation"
)

func TestNavigationReadFiltersByOptionalAdministratorSession(t *testing.T) {
	t.Parallel()

	reader := &fakeNavigationReader{groups: []navigation.Group{
		{
			Category: navigation.Category{ID: 1, Name: "Public", Slug: "public", Visibility: navigation.VisibilityPublic},
			Links:    []navigation.Link{{ID: 2, Name: "Example", URL: "https://example.com", IconSource: navigation.IconSourceGenerated}},
		},
	}}
	authenticator := &fakeAuthenticator{}
	handler := New(Config{Auth: authenticator, Navigation: reader})

	publicResponse := httptest.NewRecorder()
	handler.ServeHTTP(publicResponse, httptest.NewRequest(http.MethodGet, "/api/navigation", nil))
	if publicResponse.Code != http.StatusOK || reader.includePrivate {
		t.Fatalf("unexpected public response: status=%d private=%v", publicResponse.Code, reader.includePrivate)
	}
	if !strings.Contains(publicResponse.Body.String(), `"administrator":false`) || strings.Contains(publicResponse.Body.String(), `null`) {
		t.Fatalf("unexpected public body: %s", publicResponse.Body.String())
	}
	if cache := publicResponse.Header().Get("Cache-Control"); !strings.Contains(cache, "public") {
		t.Fatalf("unexpected public cache header %q", cache)
	}

	adminRequest := httptest.NewRequest(http.MethodGet, "/api/navigation", nil)
	adminRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	adminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminResponse, adminRequest)
	if adminResponse.Code != http.StatusOK || !reader.includePrivate {
		t.Fatalf("unexpected admin response: status=%d private=%v", adminResponse.Code, reader.includePrivate)
	}
	if !strings.Contains(adminResponse.Body.String(), `"administrator":true`) {
		t.Fatalf("unexpected admin body: %s", adminResponse.Body.String())
	}
	if adminResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected administrator navigation to disable caching")
	}
}

func TestNavigationInvalidSessionFallsBackToPublic(t *testing.T) {
	t.Parallel()

	reader := &fakeNavigationReader{}
	authenticator := &fakeAuthenticator{authenticateErr: auth.ErrUnauthenticated}
	handler := New(Config{Auth: authenticator, Navigation: reader})
	request := httptest.NewRequest(http.MethodGet, "/api/navigation", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "expired-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || reader.includePrivate {
		t.Fatalf("unexpected response: status=%d private=%v", response.Code, reader.includePrivate)
	}
	assertClearedSessionCookie(t, response)
}

func TestNavigationReadErrorsArePrivate(t *testing.T) {
	t.Parallel()

	handler := New(Config{Navigation: &fakeNavigationReader{err: errors.New("database unavailable")}})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/navigation", nil))
	if response.Code != http.StatusInternalServerError || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected error response: status=%d cache=%q", response.Code, response.Header().Get("Cache-Control"))
	}
}

type fakeNavigationReader struct {
	groups         []navigation.Group
	err            error
	includePrivate bool
}

func (reader *fakeNavigationReader) Navigation(_ context.Context, includePrivate bool) ([]navigation.Group, error) {
	reader.includePrivate = includePrivate
	return reader.groups, reader.err
}

var _ NavigationReader = (*fakeNavigationReader)(nil)
