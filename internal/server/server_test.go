package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHealthEndpoints(t *testing.T) {
	t.Parallel()

	handler := New(Config{Version: "test"})

	t.Run("liveness", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", response.Code)
		}
		if response.Body.String() != "ok\n" {
			t.Fatalf("unexpected body %q", response.Body.String())
		}
	})

	t.Run("api health", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", response.Code)
		}
		if !strings.Contains(response.Body.String(), `"version":"test"`) {
			t.Fatalf("unexpected body %q", response.Body.String())
		}
		if response.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatal("expected privacy header")
		}
		for name, want := range map[string]string{
			"Content-Security-Policy":      "frame-ancestors 'none'",
			"Cross-Origin-Opener-Policy":   "same-origin",
			"Cross-Origin-Resource-Policy": "same-origin",
			"Permissions-Policy":           "camera=()",
			"X-Content-Type-Options":       "nosniff",
			"X-Frame-Options":              "DENY",
		} {
			if !strings.Contains(response.Header().Get(name), want) {
				t.Fatalf("expected %s to contain %q, got %q", name, want, response.Header().Get(name))
			}
		}
	})
}

func TestEmbeddedSPA(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html>app shell</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ready')")},
	}
	handler := New(Config{Assets: assets, Version: "test"})

	tests := []struct {
		name          string
		path          string
		status        int
		bodyContains  string
		cacheContains string
	}{
		{name: "root", path: "/", status: http.StatusOK, bodyContains: "app shell", cacheContains: "no-cache"},
		{name: "spa fallback", path: "/settings", status: http.StatusOK, bodyContains: "app shell", cacheContains: "no-cache"},
		{name: "hashed asset", path: "/assets/app.js", status: http.StatusOK, bodyContains: "ready", cacheContains: "immutable"},
		{name: "missing asset", path: "/assets/missing.js", status: http.StatusNotFound},
		{name: "unknown API", path: "/api/unknown", status: http.StatusNotFound, cacheContains: "no-store"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			result := response.Result()
			defer result.Body.Close()
			body, err := io.ReadAll(result.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			if result.StatusCode != test.status {
				t.Fatalf("expected status %d, got %d", test.status, result.StatusCode)
			}
			if test.bodyContains != "" && !strings.Contains(string(body), test.bodyContains) {
				t.Fatalf("expected body to contain %q, got %q", test.bodyContains, string(body))
			}
			if test.cacheContains != "" && !strings.Contains(result.Header.Get("Cache-Control"), test.cacheContains) {
				t.Fatalf("expected cache header to contain %q, got %q", test.cacheContains, result.Header.Get("Cache-Control"))
			}
		})
	}
}
