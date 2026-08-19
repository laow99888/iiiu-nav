package server

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/bookmarks"
	"iiiu-nav/internal/navigation"
)

func TestBookmarkImportPreviewAndCommitEndpoints(t *testing.T) {
	t.Parallel()
	repository := &importRepository{groups: []navigation.Group{{
		Category: navigation.Category{ID: 3, Name: "工具"},
		Links:    []navigation.Link{{ID: 5, Name: "Existing", URL: "https://example.com/"}},
	}}}
	authenticator := &importAuthenticator{}
	handler := New(Config{Auth: authenticator, Imports: bookmarks.New(repository)})
	content := `<!doctype html><dl><dt><h3>工具</h3><dl><dt><a href="https://example.com#same">Same</a><dt><a href="https://new.example">New</a></dl></dl>`

	previewRequest := importRequest(t, "/api/imports/preview", content, map[string]string{"visibility": "private"})
	previewRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	previewResponse := httptest.NewRecorder()
	handler.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK || !strings.Contains(previewResponse.Body.String(), `"duplicates":1`) || !strings.Contains(previewResponse.Body.String(), `"existingId":"3"`) {
		t.Fatalf("unexpected preview: status=%d body=%s", previewResponse.Code, previewResponse.Body.String())
	}

	commitRequest := importRequest(t, "/api/imports/commit", content, map[string]string{"visibility": "public", "duplicates": "update"})
	commitRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	commitResponse := httptest.NewRecorder()
	handler.ServeHTTP(commitResponse, commitRequest)
	if commitResponse.Code != http.StatusOK || !repository.committed {
		t.Fatalf("unexpected commit: status=%d body=%s", commitResponse.Code, commitResponse.Body.String())
	}
	if repository.options.Visibility != navigation.VisibilityPublic || repository.options.Duplicates != bookmarks.DuplicateUpdate {
		t.Fatalf("unexpected options: %+v", repository.options)
	}
}

func TestBookmarkImportRejectsAnonymousInvalidAndOversizeRequests(t *testing.T) {
	t.Parallel()
	repository := &importRepository{}
	handler := New(Config{Auth: &importAuthenticator{}, Imports: bookmarks.New(repository)})

	anonymous := importRequest(t, "/api/imports/preview", `<a href="https://example.com">Example</a>`, nil)
	anonymousResponse := httptest.NewRecorder()
	handler.ServeHTTP(anonymousResponse, anonymous)
	if anonymousResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous rejection, got %d", anonymousResponse.Code)
	}

	invalid := importRequest(t, "/api/imports/preview", `<meta charset="unknown-made-up">`, nil)
	invalid.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "bookmark_charset_unsupported") {
		t.Fatalf("unexpected invalid response: %d %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	tooLarge := importRequest(t, "/api/imports/preview", strings.Repeat("x", bookmarks.MaxImportBytes+1), nil)
	tooLarge.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	tooLargeResponse := httptest.NewRecorder()
	handler.ServeHTTP(tooLargeResponse, tooLarge)
	if tooLargeResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected size rejection, got %d %s", tooLargeResponse.Code, tooLargeResponse.Body.String())
	}
}

func TestBookmarkExportEndpointsAndScopeValidation(t *testing.T) {
	t.Parallel()
	repository := &importRepository{groups: []navigation.Group{
		{Category: navigation.Category{ID: 1, Name: "Public", Visibility: navigation.VisibilityPublic}, Links: []navigation.Link{{ID: 2, Name: "Example", URL: "https://example.com", IconSource: navigation.IconSourceGenerated, IconValue: "E"}}},
		{Category: navigation.Category{ID: 3, Name: "Private", Visibility: navigation.VisibilityPrivate}},
	}}
	handler := New(Config{Auth: &importAuthenticator{}, Imports: bookmarks.New(repository)})

	htmlRequest := httptest.NewRequest(http.MethodGet, "/api/bookmarks/export?format=html&scope=public", nil)
	htmlRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	htmlResponse := httptest.NewRecorder()
	handler.ServeHTTP(htmlResponse, htmlRequest)
	if htmlResponse.Code != http.StatusOK || !strings.Contains(htmlResponse.Header().Get("Content-Type"), "text/html") || !strings.Contains(htmlResponse.Header().Get("Content-Disposition"), ".html") || strings.Contains(htmlResponse.Body.String(), "Private") {
		t.Fatalf("unexpected HTML export: status=%d headers=%v body=%s", htmlResponse.Code, htmlResponse.Header(), htmlResponse.Body.String())
	}

	jsonRequest := httptest.NewRequest(http.MethodGet, "/api/bookmarks/export?format=json&scope=all", nil)
	jsonRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	jsonResponse := httptest.NewRecorder()
	handler.ServeHTTP(jsonResponse, jsonRequest)
	if jsonResponse.Code != http.StatusOK || !strings.Contains(jsonResponse.Body.String(), `"visibility": "private"`) {
		t.Fatalf("unexpected JSON export: status=%d body=%s", jsonResponse.Code, jsonResponse.Body.String())
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/api/bookmarks/export?format=json&scope=unknown", nil)
	invalidRequest.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "active"})
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected invalid scope rejection, got %d", invalidResponse.Code)
	}

	anonymousResponse := httptest.NewRecorder()
	handler.ServeHTTP(anonymousResponse, httptest.NewRequest(http.MethodGet, "/api/bookmarks/export?format=html", nil))
	if anonymousResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous rejection, got %d", anonymousResponse.Code)
	}
}

func importRequest(t *testing.T, target, content string, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("bookmarks", "bookmarks.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, target, &body)
	request.Host = "nav.test"
	request.Header.Set("Origin", "http://nav.test")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

type importRepository struct {
	groups    []navigation.Group
	options   bookmarks.ImportOptions
	committed bool
}

func (repository *importRepository) Navigation(context.Context, bool) ([]navigation.Group, error) {
	return repository.groups, nil
}
func (repository *importRepository) ImportBookmarks(_ context.Context, batch bookmarks.Batch, options bookmarks.ImportOptions) (bookmarks.CommitResult, error) {
	repository.options = options
	repository.committed = true
	return bookmarks.CommitResult{CreatedLinks: len(batch.Categories[0].Links)}, nil
}

type importAuthenticator struct{}

func (*importAuthenticator) Login(context.Context, string) (auth.SessionToken, error) {
	return auth.SessionToken{}, nil
}
func (*importAuthenticator) Authenticate(_ context.Context, token string) error {
	if token == "" {
		return auth.ErrUnauthenticated
	}
	return nil
}
func (*importAuthenticator) Logout(context.Context, string) error { return nil }
func (*importAuthenticator) ChangePassword(context.Context, string, string, string) error {
	return errors.New("not implemented")
}
