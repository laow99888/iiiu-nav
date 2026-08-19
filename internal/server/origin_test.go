package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOriginOnly(t *testing.T) {
	t.Parallel()

	handler := sameOriginOnly(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	tests := []struct {
		name       string
		method     string
		origin     string
		referer    string
		fetchSite  string
		forwarded  string
		wantStatus int
	}{
		{name: "safe method", method: http.MethodGet, wantStatus: http.StatusNoContent},
		{name: "matching origin", method: http.MethodPost, origin: "http://nav.test", wantStatus: http.StatusNoContent},
		{name: "matching referer", method: http.MethodPost, referer: "http://nav.test/settings", wantStatus: http.StatusNoContent},
		{name: "proxied HTTPS", method: http.MethodPost, origin: "https://nav.test", forwarded: "https", wantStatus: http.StatusNoContent},
		{name: "missing source", method: http.MethodPost, wantStatus: http.StatusForbidden},
		{name: "wrong host", method: http.MethodPost, origin: "http://nav.test.attacker.example", wantStatus: http.StatusForbidden},
		{name: "wrong scheme", method: http.MethodPost, origin: "https://nav.test", wantStatus: http.StatusForbidden},
		{name: "cross site metadata", method: http.MethodPost, origin: "http://nav.test", fetchSite: "cross-site", wantStatus: http.StatusForbidden},
		{name: "same site metadata", method: http.MethodPost, origin: "http://nav.test", fetchSite: "same-site", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/api/test", nil)
			request.Host = "nav.test"
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Referer", test.referer)
			request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			request.Header.Set("X-Forwarded-Proto", test.forwarded)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d body=%s", test.wantStatus, response.Code, response.Body.String())
			}
			if !isSafeMethod(test.method) && response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("expected unsafe response to be non-cacheable, got %q", response.Header().Get("Cache-Control"))
			}
		})
	}
}
