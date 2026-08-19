package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/linkmeta"
	"iiiu-nav/internal/navigation"
	"iiiu-nav/internal/server"
	storage "iiiu-nav/internal/storage/sqlite"
)

func TestAdministratorHTTPWorkflow(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "nav.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})

	manager, err := auth.New(auth.Config{
		Store:    store,
		Password: auth.PasswordParams{Memory: 7 * 1024, Time: 1, Threads: 1},
	})
	if err != nil {
		t.Fatalf("create authentication manager: %v", err)
	}
	passwordFile := filepath.Join(t.TempDir(), "admin-password")
	if err := os.WriteFile(passwordFile, []byte("original administrator password\n"), 0o600); err != nil {
		t.Fatalf("write password file: %v", err)
	}
	created, err := manager.BootstrapFromFile(ctx, passwordFile)
	if err != nil || !created {
		t.Fatalf("bootstrap administrator: created=%v err=%v", created, err)
	}

	testServer := httptest.NewServer(server.New(server.Config{Auth: manager, Version: "test"}))
	t.Cleanup(testServer.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	response := postJSON(t, client, testServer.URL+"/api/auth/login", testServer.URL, map[string]string{
		"password": "original administrator password",
	})
	assertStatus(t, response, http.StatusNoContent)

	authenticated := sessionStatus(t, client, testServer.URL)
	if !authenticated {
		t.Fatal("expected authenticated session")
	}

	response = postJSON(t, client, testServer.URL+"/api/auth/password", testServer.URL, map[string]string{
		"currentPassword": "original administrator password",
		"newPassword":     "newpass09",
	})
	assertStatus(t, response, http.StatusNoContent)
	if sessionStatus(t, client, testServer.URL) {
		t.Fatal("password change did not clear the active session")
	}

	response = postJSON(t, client, testServer.URL+"/api/auth/login", testServer.URL, map[string]string{
		"password": "original administrator password",
	})
	assertStatus(t, response, http.StatusUnauthorized)
	response = postJSON(t, client, testServer.URL+"/api/auth/login", testServer.URL, map[string]string{
		"password": "newpass09",
	})
	assertStatus(t, response, http.StatusNoContent)

	logoutRequest, err := http.NewRequest(http.MethodPost, testServer.URL+"/api/auth/logout", nil)
	if err != nil {
		t.Fatalf("create logout request: %v", err)
	}
	logoutRequest.Header.Set("Origin", testServer.URL)
	response, err = client.Do(logoutRequest)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	assertStatus(t, response, http.StatusNoContent)
	if sessionStatus(t, client, testServer.URL) {
		t.Fatal("logout did not invalidate the session")
	}

	response = postJSON(t, client, testServer.URL+"/api/auth/register", testServer.URL, map[string]string{
		"password": "another administrator password",
	})
	assertStatus(t, response, http.StatusNotFound)
}

func TestNavigationHTTPPrivacyWorkflow(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "nav.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})

	publicCategory, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name: "公开", Slug: "public", Visibility: navigation.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("create public category: %v", err)
	}
	privateCategory, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name: "私有", Slug: "private", Visibility: navigation.VisibilityPrivate, SortOrder: 10,
	})
	if err != nil {
		t.Fatalf("create private category: %v", err)
	}
	for _, input := range []navigation.LinkInput{
		{CategoryID: publicCategory.ID, Name: "Public link", URL: "https://public.example", IconSource: navigation.IconSourceGenerated},
		{CategoryID: privateCategory.ID, Name: "Private link", URL: "https://private.example", IconSource: navigation.IconSourceGenerated},
	} {
		if _, err := store.CreateLink(ctx, input); err != nil {
			t.Fatalf("create link %q: %v", input.Name, err)
		}
	}

	manager, err := auth.New(auth.Config{
		Store: store, Password: auth.PasswordParams{Memory: 7 * 1024, Time: 1, Threads: 1},
	})
	if err != nil {
		t.Fatalf("create authentication manager: %v", err)
	}
	passwordFile := filepath.Join(t.TempDir(), "admin-password")
	if err := os.WriteFile(passwordFile, []byte("navigation administrator password\n"), 0o600); err != nil {
		t.Fatalf("write password file: %v", err)
	}
	if created, err := manager.BootstrapFromFile(ctx, passwordFile); err != nil || !created {
		t.Fatalf("bootstrap administrator: created=%v err=%v", created, err)
	}

	logoRoot := filepath.Join(t.TempDir(), "logos")
	if err := os.MkdirAll(logoRoot, 0o700); err != nil {
		t.Fatalf("create logo root: %v", err)
	}
	metadataSource := &metadataTestSource{icon: testPNGBytes(t)}
	recognizer := linkmeta.NewWithClient(metadataSource, func(_ context.Context, target *url.URL) error {
		if target.Hostname() == "127.0.0.1" || target.Hostname() == "localhost" {
			return linkmeta.ErrUnsafeURL
		}
		return nil
	})
	testServer := httptest.NewServer(server.New(server.Config{
		Auth: manager, Categories: store, Links: store, Logos: imagestore.NewLogos(logoRoot), Metadata: recognizer, Navigation: store, Settings: store, Version: "test",
	}))
	t.Cleanup(testServer.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	public := readNavigation(t, client, testServer.URL)
	if public.Administrator || len(public.Categories) != 1 || public.Categories[0].Name != "公开" {
		t.Fatalf("unexpected public navigation: %+v", public)
	}
	if public.Categories[0].Links[0].Name != "Public link" {
		t.Fatalf("unexpected public links: %+v", public.Categories[0].Links)
	}

	response := postJSON(t, client, testServer.URL+"/api/auth/login", testServer.URL, map[string]string{
		"password": "navigation administrator password",
	})
	assertStatus(t, response, http.StatusNoContent)
	administrator := readNavigation(t, client, testServer.URL)
	if !administrator.Administrator || len(administrator.Categories) != 2 {
		t.Fatalf("unexpected administrator navigation: %+v", administrator)
	}
	if administrator.Categories[1].Visibility != "private" || administrator.Categories[1].Links[0].Name != "Private link" {
		t.Fatalf("private navigation missing: %+v", administrator.Categories[1])
	}
	uploadedLogo := uploadTestLogo(t, client, testServer.URL)
	logoResponse, err := client.Get(testServer.URL + uploadedLogo)
	if err != nil {
		t.Fatalf("read uploaded logo: %v", err)
	}
	if logoResponse.StatusCode != http.StatusOK || logoResponse.Header.Get("Content-Type") != "image/png" || !strings.Contains(logoResponse.Header.Get("Cache-Control"), "immutable") {
		logoResponse.Body.Close()
		t.Fatalf("unexpected logo response: status=%d headers=%v", logoResponse.StatusCode, logoResponse.Header)
	}
	logoResponse.Body.Close()
	response = postJSON(t, client, testServer.URL+"/api/metadata/recognize", testServer.URL, map[string]string{"url": testServer.URL})
	if response.StatusCode != http.StatusUnprocessableEntity {
		content, _ := io.ReadAll(response.Body)
		response.Body.Close()
		t.Fatalf("recognize unsafe metadata status=%d body=%s", response.StatusCode, content)
	}
	var metadataFailure map[string]string
	if err := json.NewDecoder(response.Body).Decode(&metadataFailure); err != nil {
		response.Body.Close()
		t.Fatalf("decode metadata failure: %v", err)
	}
	response.Body.Close()
	if metadataFailure["error"] != "metadata_target_not_public" {
		t.Fatalf("unexpected metadata failure: %+v", metadataFailure)
	}
	response = postJSON(t, client, testServer.URL+"/api/metadata/recognize", testServer.URL, map[string]string{"url": "https://recognized.example/page"})
	if response.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(response.Body)
		response.Body.Close()
		t.Fatalf("recognize metadata status=%d body=%s", response.StatusCode, content)
	}
	var recognized recognitionHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&recognized); err != nil {
		response.Body.Close()
		t.Fatalf("decode recognized metadata: %v", err)
	}
	response.Body.Close()
	if recognized.Name != "Recognized site" || !strings.HasPrefix(recognized.IconValue, imagestore.LogoPrefix) {
		t.Fatalf("unexpected recognized metadata: %+v", recognized)
	}

	response = doJSON(t, client, http.MethodPost, testServer.URL+"/api/links", testServer.URL, map[string]any{
		"categoryId": strconv.FormatInt(publicCategory.ID, 10), "name": "Created link",
		"description": "Created through HTTP", "url": "https://recognized.example/page", "iconSource": "auto", "iconValue": recognized.IconValue,
	})
	assertStatus(t, response, http.StatusCreated)
	afterLinkCreate := readNavigation(t, client, testServer.URL)
	if len(afterLinkCreate.Categories[0].Links) != 2 {
		t.Fatalf("created link missing: %+v", afterLinkCreate.Categories[0])
	}
	createdLinkID := afterLinkCreate.Categories[0].Links[1].ID
	metadataSource.failed.Store(true)
	response = doJSON(t, client, http.MethodPost, testServer.URL+"/api/links/"+createdLinkID+"/refresh", testServer.URL, nil)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	if _, err := os.Stat(filepath.Join(logoRoot, filepath.Base(recognized.IconValue))); err != nil {
		t.Fatalf("cached logo disappeared after recognition failure: %v", err)
	}
	metadataSource.failed.Store(false)
	response = doJSON(t, client, http.MethodPut, testServer.URL+"/api/categories/"+strconv.FormatInt(publicCategory.ID, 10)+"/links/order", testServer.URL, map[string]any{
		"ids": []string{createdLinkID, afterLinkCreate.Categories[0].Links[0].ID},
	})
	assertStatus(t, response, http.StatusNoContent)
	if afterOrder := readNavigation(t, client, testServer.URL); afterOrder.Categories[0].Links[0].ID != createdLinkID {
		t.Fatalf("link order did not persist: %+v", afterOrder.Categories[0].Links)
	}
	response = doJSON(t, client, http.MethodPut, testServer.URL+"/api/links/"+createdLinkID, testServer.URL, map[string]any{
		"categoryId": strconv.FormatInt(privateCategory.ID, 10), "name": "Moved link",
		"description": "Moved through HTTP", "url": "https://moved.example/", "logoText": "ML",
	})
	assertStatus(t, response, http.StatusOK)
	if _, err := os.Stat(filepath.Join(logoRoot, filepath.Base(recognized.IconValue))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replaced unreferenced logo was not removed: %v", err)
	}
	if afterMove := readNavigation(t, client, testServer.URL); len(afterMove.Categories[1].Links) != 2 {
		t.Fatalf("moved link missing: %+v", afterMove.Categories[1].Links)
	}
	deleteLinkRequest, err := http.NewRequest(http.MethodDelete, testServer.URL+"/api/links/"+createdLinkID, nil)
	if err != nil {
		t.Fatalf("create link delete request: %v", err)
	}
	deleteLinkRequest.Header.Set("Origin", testServer.URL)
	response, err = client.Do(deleteLinkRequest)
	if err != nil {
		t.Fatalf("delete link: %v", err)
	}
	assertStatus(t, response, http.StatusNoContent)

	response = doJSON(t, client, http.MethodPost, testServer.URL+"/api/categories", testServer.URL, map[string]any{
		"name": "新增私有", "iconName": "code", "visibility": "private",
	})
	assertStatus(t, response, http.StatusCreated)
	afterCreate := readNavigation(t, client, testServer.URL)
	if len(afterCreate.Categories) != 3 || afterCreate.Categories[2].Name != "新增私有" {
		t.Fatalf("created category missing after reload: %+v", afterCreate)
	}
	createdID := afterCreate.Categories[2].ID
	response = doJSON(t, client, http.MethodPut, testServer.URL+"/api/categories/"+createdID, testServer.URL, map[string]any{
		"name": "已修改私有", "iconName": "star", "visibility": "private",
	})
	assertStatus(t, response, http.StatusOK)
	response = doJSON(t, client, http.MethodPut, testServer.URL+"/api/categories/order", testServer.URL, map[string]any{
		"ids": []string{createdID, afterCreate.Categories[0].ID, afterCreate.Categories[1].ID},
	})
	assertStatus(t, response, http.StatusNoContent)
	afterReorder := readNavigation(t, client, testServer.URL)
	if afterReorder.Categories[0].ID != createdID || afterReorder.Categories[0].Name != "已修改私有" {
		t.Fatalf("updated category order did not persist: %+v", afterReorder)
	}
	deleteRequest, err := http.NewRequest(http.MethodDelete, testServer.URL+"/api/categories/"+createdID+"?links=delete", nil)
	if err != nil {
		t.Fatalf("create category delete request: %v", err)
	}
	deleteRequest.Header.Set("Origin", testServer.URL)
	response, err = client.Do(deleteRequest)
	if err != nil {
		t.Fatalf("delete category: %v", err)
	}
	assertStatus(t, response, http.StatusNoContent)
	if afterDelete := readNavigation(t, client, testServer.URL); len(afterDelete.Categories) != 2 {
		t.Fatalf("deleted category remained: %+v", afterDelete)
	}

	response = doJSON(t, client, http.MethodPost, testServer.URL+"/api/auth/logout", testServer.URL, nil)
	assertStatus(t, response, http.StatusNoContent)
	publicAfterMutation := readNavigation(t, client, testServer.URL)
	if len(publicAfterMutation.Categories) != 1 || publicAfterMutation.Categories[0].Name != "公开" {
		t.Fatalf("private mutation leaked publicly: %+v", publicAfterMutation)
	}

	response = doJSON(t, client, http.MethodPost, testServer.URL+"/api/categories", testServer.URL, map[string]any{
		"name": "Anonymous", "iconName": "", "visibility": "public",
	})
	assertStatus(t, response, http.StatusUnauthorized)
}

type recognitionHTTPResponse struct {
	Name       string `json:"name"`
	IconSource string `json:"iconSource"`
	IconValue  string `json:"iconValue"`
}

type metadataTestSource struct {
	failed atomic.Bool
	icon   []byte
}

func (source *metadataTestSource) Do(request *http.Request) (*http.Response, error) {
	if source.failed.Load() {
		return nil, errors.New("source unavailable")
	}
	header := make(http.Header)
	var content []byte
	if request.URL.Path == "/favicon.ico" {
		header.Set("Content-Type", "image/png")
		content = source.icon
	} else {
		header.Set("Content-Type", "text/html; charset=utf-8")
		content = []byte(`<html><head><title>Recognized site</title><meta name="description" content="Recognized description"></head></html>`)
	}
	return &http.Response{
		StatusCode: http.StatusOK, Header: header,
		Body: io.NopCloser(bytes.NewReader(content)), Request: request,
	}, nil
}

func testPNGBytes(t *testing.T) []byte {
	t.Helper()
	var content bytes.Buffer
	fixture := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	fixture.Set(0, 0, color.NRGBA{R: 220, G: 40, B: 40, A: 255})
	if err := png.Encode(&content, fixture); err != nil {
		t.Fatalf("encode test logo: %v", err)
	}
	return content.Bytes()
}

func uploadTestLogo(t *testing.T, client *http.Client, baseURL string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("logo", "client-name.png")
	if err != nil {
		t.Fatalf("create logo form: %v", err)
	}
	if _, err := part.Write(testPNGBytes(t)); err != nil {
		t.Fatalf("write logo form: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close logo form: %v", err)
	}
	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/logos", &body)
	if err != nil {
		t.Fatalf("create logo request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", baseURL)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("upload logo: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		content, _ := io.ReadAll(response.Body)
		t.Fatalf("upload logo status=%d body=%s", response.StatusCode, content)
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode uploaded logo: %v", err)
	}
	return result.URL
}

type navigationHTTPResponse struct {
	Administrator bool `json:"administrator"`
	Categories    []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Visibility string `json:"visibility"`
		Links      []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"links"`
	} `json:"categories"`
}

func readNavigation(t *testing.T, client *http.Client, baseURL string) navigationHTTPResponse {
	t.Helper()
	response, err := client.Get(baseURL + "/api/navigation")
	if err != nil {
		t.Fatalf("read navigation: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("navigation status=%d body=%s", response.StatusCode, body)
	}
	var result navigationHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode navigation: %v", err)
	}
	return result
}

func postJSON(t *testing.T, client *http.Client, target, origin string, value any) *http.Response {
	return doJSON(t, client, http.MethodPost, target, origin, value)
}

func doJSON(t *testing.T, client *http.Client, method, target, origin string, value any) *http.Response {
	t.Helper()
	var body []byte
	var err error
	if value != nil {
		body, err = json.Marshal(value)
	}
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	request, err := http.NewRequest(method, target, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if value != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Origin", origin)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return response
}

func sessionStatus(t *testing.T, client *http.Client, baseURL string) bool {
	t.Helper()
	response, err := client.Get(baseURL + "/api/auth/session")
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("session status=%d body=%s", response.StatusCode, body)
	}
	var result struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return result.Authenticated
}

func assertStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode != want {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d body=%s", want, response.StatusCode, body)
	}
}
