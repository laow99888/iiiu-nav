package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iiiu-nav/internal/analytics"
	"iiiu-nav/internal/auth"
)

func TestAnalyticsRouteRequiresAdministrator(t *testing.T) {
	t.Parallel()
	service := &fakePageViewAnalytics{series: []analytics.DailyPageViews{{Day: "2026-08-19", Views: 7}}}
	authenticator := &fakeAuthenticator{}
	mux := http.NewServeMux()
	registerAnalyticsRoutes(mux, authenticator, service)

	request := httptest.NewRequest(http.MethodGet, "/api/analytics/page-views", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "valid"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"views":7`) {
		t.Fatalf("unexpected administrator response %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected analytics response to disable caching")
	}

	authenticator.authenticateErr = auth.ErrUnauthenticated
	anonymous := httptest.NewRecorder()
	mux.ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/api/analytics/page-views", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous status 401, got %d", anonymous.Code)
	}
}

func TestPublicPageViewTrackingFiltersRequests(t *testing.T) {
	t.Parallel()
	service := &fakePageViewAnalytics{}
	authenticator := &fakeAuthenticator{authenticateErr: auth.ErrUnauthenticated}
	handler := trackPublicPageViews(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}), authenticator, service)

	for _, test := range []struct {
		name      string
		method    string
		path      string
		userAgent string
		counted   bool
	}{
		{name: "public browser", method: http.MethodGet, path: "/", userAgent: "Mozilla/5.0", counted: true},
		{name: "admin route", method: http.MethodGet, path: "/admin", userAgent: "Mozilla/5.0"},
		{name: "head", method: http.MethodHead, path: "/", userAgent: "Mozilla/5.0"},
		{name: "robot", method: http.MethodGet, path: "/", userAgent: "ExampleBot/1.0"},
		{name: "command line", method: http.MethodGet, path: "/", userAgent: "curl/8.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := service.recorded
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("User-Agent", test.userAgent)
			handler.ServeHTTP(httptest.NewRecorder(), request)
			if got := service.recorded > before; got != test.counted {
				t.Fatalf("counted=%t want %t", got, test.counted)
			}
		})
	}

	authenticator.authenticateErr = nil
	admin := httptest.NewRequest(http.MethodGet, "/", nil)
	admin.Header.Set("User-Agent", "Mozilla/5.0")
	admin.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "valid"})
	before := service.recorded
	handler.ServeHTTP(httptest.NewRecorder(), admin)
	if service.recorded != before {
		t.Fatal("administrator page load was counted")
	}
}

type fakePageViewAnalytics struct {
	recorded int
	series   []analytics.DailyPageViews
}

func (service *fakePageViewAnalytics) RecordPageView(context.Context) error {
	service.recorded++
	return nil
}

func (service *fakePageViewAnalytics) PageViewSeries(context.Context, int) ([]analytics.DailyPageViews, error) {
	if service.series == nil {
		return nil, errors.New("series unavailable")
	}
	return service.series, nil
}
