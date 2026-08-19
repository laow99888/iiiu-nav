package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"iiiu-nav/internal/auth"
)

func TestLoginSetsProtectedSessionCookie(t *testing.T) {
	t.Parallel()

	fake := &fakeAuthenticator{loginToken: auth.SessionToken{
		Value:     "session-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}}
	handler := New(Config{Auth: fake})
	request := authRequest(http.MethodPost, "/api/auth/login", `{"password":"correct password"}`)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", response.Code, response.Body.String())
	}
	if fake.loginPassword != "correct password" {
		t.Fatalf("unexpected login password %q", fake.loginPassword)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName || cookie.Value != "session-token" || cookie.Path != "/" {
		t.Fatalf("unexpected cookie: %+v", cookie)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure {
		t.Fatalf("unexpected cookie security attributes: %+v", cookie)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected private response to disable caching")
	}
}

func TestLoginSetsSecureCookieBehindHTTPSProxy(t *testing.T) {
	t.Parallel()

	fake := &fakeAuthenticator{loginToken: auth.SessionToken{Value: "secure-token", ExpiresAt: time.Now().Add(time.Hour)}}
	handler := New(Config{Auth: fake})
	request := authRequest(http.MethodPost, "/api/auth/login", `{"password":"correct password"}`)
	request.Header.Set("Origin", "https://nav.test")
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", response.Code, response.Body.String())
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("expected secure cookie, got %+v", cookies)
	}
}

func TestLoginRateLimit(t *testing.T) {
	t.Parallel()

	fake := &fakeAuthenticator{loginErr: auth.ErrInvalidCredentials}
	handler := New(Config{Auth: fake})

	for attempt := 1; attempt <= 6; attempt++ {
		request := authRequest(http.MethodPost, "/api/auth/login", `{"password":"incorrect password"}`)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		want := http.StatusUnauthorized
		if attempt == 6 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("attempt %d: expected status %d, got %d", attempt, want, response.Code)
		}
		if attempt == 6 && response.Header().Get("Retry-After") == "" {
			t.Fatal("expected Retry-After header")
		}
	}
	if fake.loginCalls != 5 {
		t.Fatalf("expected five password checks, got %d", fake.loginCalls)
	}
}

func TestSessionLogoutAndPasswordChange(t *testing.T) {
	t.Parallel()

	fake := &fakeAuthenticator{}
	handler := New(Config{Auth: fake})

	sessionRequest := authRequest(http.MethodGet, "/api/auth/session", "")
	sessionRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	sessionResponse := httptest.NewRecorder()
	handler.ServeHTTP(sessionResponse, sessionRequest)
	if sessionResponse.Code != http.StatusOK || !strings.Contains(sessionResponse.Body.String(), `"authenticated":true`) {
		t.Fatalf("unexpected session response: status=%d body=%s", sessionResponse.Code, sessionResponse.Body.String())
	}

	logoutRequest := authRequest(http.MethodPost, "/api/auth/logout", "")
	logoutRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent || fake.logoutToken != "active-token" {
		t.Fatalf("unexpected logout: status=%d token=%q", logoutResponse.Code, fake.logoutToken)
	}
	assertClearedSessionCookie(t, logoutResponse)

	passwordRequest := authRequest(
		http.MethodPost,
		"/api/auth/password",
		`{"currentPassword":"current password","newPassword":"replacement password"}`,
	)
	passwordRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	passwordResponse := httptest.NewRecorder()
	handler.ServeHTTP(passwordResponse, passwordRequest)
	if passwordResponse.Code != http.StatusNoContent {
		t.Fatalf("unexpected password response: status=%d body=%s", passwordResponse.Code, passwordResponse.Body.String())
	}
	if fake.changedToken != "active-token" || fake.currentPassword != "current password" || fake.newPassword != "replacement password" {
		t.Fatalf("unexpected password change values: %+v", fake)
	}
	assertClearedSessionCookie(t, passwordResponse)
}

func TestRequireAdminRejectsAnonymousRequests(t *testing.T) {
	t.Parallel()

	fake := &fakeAuthenticator{authenticateErr: auth.ErrUnauthenticated}
	protected := RequireAdmin(fake, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/private", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}

	fake.authenticateErr = nil
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected authenticated status 204, got %d", response.Code)
	}
}

func authRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Host = "nav.test"
	request.RemoteAddr = "192.0.2.10:12345"
	request.Header.Set("Origin", "http://nav.test")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func assertClearedSessionCookie(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != SessionCookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared session cookie, got %+v", cookies)
	}
}

type fakeAuthenticator struct {
	loginToken      auth.SessionToken
	loginErr        error
	loginPassword   string
	loginCalls      int
	authenticateErr error
	authenticate    string
	logoutToken     string
	changedToken    string
	currentPassword string
	newPassword     string
	changeErr       error
}

func (fake *fakeAuthenticator) Login(_ context.Context, password string) (auth.SessionToken, error) {
	fake.loginCalls++
	fake.loginPassword = password
	return fake.loginToken, fake.loginErr
}

func (fake *fakeAuthenticator) Authenticate(_ context.Context, token string) error {
	fake.authenticate = token
	return fake.authenticateErr
}

func (fake *fakeAuthenticator) Logout(_ context.Context, token string) error {
	fake.logoutToken = token
	return nil
}

func (fake *fakeAuthenticator) ChangePassword(_ context.Context, token, currentPassword, newPassword string) error {
	fake.changedToken = token
	fake.currentPassword = currentPassword
	fake.newPassword = newPassword
	return fake.changeErr
}

var _ Authenticator = (*fakeAuthenticator)(nil)
