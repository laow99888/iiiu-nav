package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/siteconfig"
)

func TestSiteSettingsDefaultsAndIndexingHeaders(t *testing.T) {
	t.Parallel()
	settings := &keyedSettingsStore{values: make(map[string]json.RawMessage)}
	handler := New(Config{Settings: settings, Version: "test"})

	request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("robots route should require complete site configuration, got %d", response.Code)
	}
	if response.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Fatalf("default indexing header missing: %v", response.Header())
	}

	defaults, err := readSiteSettings(request.Context(), settings)
	if err != nil || defaults != siteconfig.Default() {
		t.Fatalf("unexpected defaults: %+v err=%v", defaults, err)
	}
}

func TestSiteSettingsUpdateUploadsAndRobots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	siteRoot, backgroundRoot := filepath.Join(root, "site"), filepath.Join(root, "backgrounds")
	for _, directory := range []string{siteRoot, backgroundRoot} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	settings := &keyedSettingsStore{values: make(map[string]json.RawMessage)}
	handler := New(Config{
		Auth: &fakeAuthenticator{}, Settings: settings,
		Navigation: &fakeNavigationReader{},
		SiteImages: imagestore.NewSiteLogo(siteRoot), Favicons: imagestore.NewFavicon(siteRoot),
		Backgrounds: imagestore.NewBackgrounds(backgroundRoot), Version: "test",
	})

	logoURL := uploadSettingImage(t, handler, "/api/settings/site/logo", pngFixture(t), "logo.png")
	backgroundURL := uploadSettingImage(t, handler, "/api/settings/site/background", pngFixture(t), "background.png")
	input := siteconfig.Settings{
		Name: "My Navigation", LogoURL: logoURL, AccentColor: "#2f6f55",
		BackgroundURL: backgroundURL, BackgroundOverlay: 72, IndexingEnabled: true,
	}
	content, _ := json.Marshal(input)
	request := authenticatedRequest(http.MethodPut, "/api/settings/site", bytes.NewReader(content), "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("update site settings status=%d body=%s", response.Code, response.Body.String())
	}

	robots := httptest.NewRecorder()
	handler.ServeHTTP(robots, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	if robots.Code != http.StatusOK || !strings.Contains(robots.Body.String(), "Allow: /") {
		t.Fatalf("unexpected robots response: status=%d body=%s", robots.Code, robots.Body.String())
	}
	if robots.Header().Get("X-Robots-Tag") != "" {
		t.Fatalf("indexing header remained enabled: %v", robots.Header())
	}

	navigation := httptest.NewRecorder()
	handler.ServeHTTP(navigation, httptest.NewRequest(http.MethodGet, "/api/navigation", nil))
	if navigation.Code != http.StatusOK || !strings.Contains(navigation.Body.String(), `"name":"My Navigation"`) {
		t.Fatalf("unexpected navigation status %d", navigation.Code)
	}
}

func TestSiteSettingsRejectInvalidUploadAndAnonymousMutation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	settings := &keyedSettingsStore{values: make(map[string]json.RawMessage)}
	authenticator := &fakeAuthenticator{authenticateErr: auth.ErrUnauthenticated}
	handler := New(Config{
		Auth: authenticator, Settings: settings,
		SiteImages: imagestore.NewSiteLogo(root), Favicons: imagestore.NewFavicon(root),
		Backgrounds: imagestore.NewBackgrounds(root),
	})
	request := authenticatedRequest(http.MethodPut, "/api/settings/site", strings.NewReader(`{}`), "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous update rejection, got %d", response.Code)
	}

	authenticator.authenticateErr = nil
	invalid := uploadRequest(t, "/api/settings/site/favicon", []byte("not an image"), "fake.png")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected invalid image rejection, got %d", invalidResponse.Code)
	}
}

func uploadSettingImage(t *testing.T, handler http.Handler, path string, content []byte, name string) string {
	t.Helper()
	request := uploadRequest(t, path, content, name)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload %s status=%d body=%s", path, response.Code, response.Body.String())
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result.URL
}

func uploadRequest(t *testing.T, path string, content []byte, name string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return authenticatedRequest(http.MethodPost, path, &body, writer.FormDataContentType())
}

func authenticatedRequest(method, path string, body io.Reader, contentType string) *http.Request {
	request := httptest.NewRequest(method, path, body)
	request.Host = "nav.test"
	request.Header.Set("Origin", "http://nav.test")
	request.Header.Set("Content-Type", contentType)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active-token"})
	return request
}

func pngFixture(t *testing.T) []byte {
	t.Helper()
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewNRGBA(image.Rect(0, 0, 32, 24))); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}

type keyedSettingsStore struct{ values map[string]json.RawMessage }

func (store *keyedSettingsStore) Setting(_ context.Context, key string) (json.RawMessage, bool, error) {
	value, found := store.values[key]
	return append(json.RawMessage(nil), value...), found, nil
}

func (store *keyedSettingsStore) PutSetting(_ context.Context, key string, value json.RawMessage) error {
	store.values[key] = append(json.RawMessage(nil), value...)
	return nil
}
