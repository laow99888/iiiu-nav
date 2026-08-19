package bookmarks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

	"iiiu-nav/internal/navigation"
)

func TestParseBookmarkHTMLFoldersEncodingAndInvalidEntries(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE NETSCAPE-Bookmark-file-1><META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=gbk"><DL><p>
<DT><H3> 工具 </H3><DL><p><DT><A HREF="https://EXAMPLE.com:443#top">示例</A>
<DT><H3>开发</H3><DL><p><DT><A HREF="https://dev.example.com/path/">开发站</A></DL></DL>
<DT><A HREF="javascript:alert(1)">坏链接</A></DL>`
	encoded, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	batch, err := Parse(strings.NewReader(string(encoded)))
	if err != nil {
		t.Fatalf("parse HTML: %v", err)
	}
	if batch.Format != FormatHTML || len(batch.Categories) != 3 {
		t.Fatalf("unexpected batch: %+v", batch)
	}
	if batch.Categories[0].Name != "工具" || batch.Categories[1].Name != "工具 / 开发" || batch.Categories[2].Name != UnfiledName {
		t.Fatalf("unexpected category flattening: %+v", batch.Categories)
	}
	if batch.Categories[0].Links[0].IconValue != "示" || batch.Categories[2].Links[0].Problem != "url_invalid" {
		t.Fatalf("unexpected links: %+v", batch.Categories)
	}
}

func TestParseApplicationJSONAndUTF8BOM(t *testing.T) {
	t.Parallel()
	document := "\ufeff" + `{"format":"iiiu-nav-bookmarks","version":1,"categories":[{"name":" Cafe\u0301 ","iconName":"book","visibility":"public","links":[{"name":" Docs ","description":"Reference","url":"https://example.com","iconSource":"generated","iconValue":"D"}]}]}`
	batch, err := Parse(strings.NewReader(document))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if batch.Format != FormatJSON || batch.Categories[0].Name != "Caf\u00e9" || batch.Categories[0].Links[0].Name != "Docs" {
		t.Fatalf("JSON was not normalized: %+v", batch)
	}
}

func TestApplicationJSONMergesCaseEquivalentCategories(t *testing.T) {
	t.Parallel()
	document := `{"format":"iiiu-nav-bookmarks","version":1,"categories":[{"name":"Caf\u00e9","links":[{"name":"One","url":"https://one.example"}]},{"name":"CAFE\u0301","links":[{"name":"Two","url":"https://two.example"}]}]}`
	batch, err := Parse(strings.NewReader(document))
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Categories) != 1 || len(batch.Categories[0].Links) != 2 {
		t.Fatalf("equivalent categories were not merged: %+v", batch.Categories)
	}
}

func TestParseRejectsUnsupportedCharsetAndOversize(t *testing.T) {
	t.Parallel()
	_, err := Parse(strings.NewReader(`<meta charset="made-up"><a href="https://example.com">Example</a>`))
	if !errors.Is(err, ErrUnsupportedCharset) {
		t.Fatalf("expected charset error, got %v", err)
	}
	_, err = Parse(strings.NewReader(strings.Repeat("x", MaxImportBytes+1)))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestCanonicalURLDuplicateRules(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"HTTPS://Example.COM:443#section", "https://example.com/"},
		{"http://Example.com:80?q=1", "http://example.com/?q=1"},
	}
	for _, pair := range pairs {
		if CanonicalURL(pair[0]) != pair[1] {
			t.Fatalf("canonical URL mismatch: %q -> %q", pair[0], CanonicalURL(pair[0]))
		}
	}
	if CanonicalURL("https://example.com/path/") == CanonicalURL("https://example.com/path") {
		t.Fatal("non-root trailing slash must be preserved")
	}
}

func TestPreviewMapsCategoriesAndDuplicates(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{groups: []navigation.Group{{
		Category: navigation.Category{ID: 7, Name: "工具"},
		Links:    []navigation.Link{{ID: 9, Name: "Existing", URL: "https://example.com/"}},
	}}}
	service := New(repository)
	preview, err := service.Preview(context.Background(), strings.NewReader(`<!doctype html><dl><dt><h3>工具</h3><dl><dt><a href="HTTPS://EXAMPLE.COM:443#x">Same</a><dt><a href="https://new.example">New</a><dt><a href="https://new.example#again">Again</a></dl></dl>`), navigation.VisibilityPrivate)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Summary.Duplicates != 2 || preview.Summary.New != 1 || preview.Summary.ExistingCategories != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.Categories[0].ExistingID != "7" || preview.Categories[0].Links[0].DuplicateOf != "Existing" {
		t.Fatalf("mapping missing: %+v", preview.Categories[0])
	}
}

func TestExportsRoundTripAndFilterScopes(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{groups: []navigation.Group{
		{Category: navigation.Category{ID: 1, Name: "公开 & 工具", IconName: "tools", Visibility: navigation.VisibilityPublic}, Links: []navigation.Link{{ID: 2, Name: "Docs <One>", Description: "Reference & notes", URL: "https://example.com/?a=1&b=2", IconSource: navigation.IconSourceGenerated, IconValue: "D", CreatedAt: time.Unix(123, 0)}}},
		{Category: navigation.Category{ID: 3, Name: "私有", IconName: "lock", Visibility: navigation.VisibilityPrivate}, Links: []navigation.Link{{ID: 4, Name: "Internal", URL: "https://private.example", IconSource: navigation.IconSourceGenerated, IconValue: "I"}}},
	}}
	service := New(repository)
	htmlExport, err := service.ExportHTML(context.Background(), ScopePublic)
	if err != nil {
		t.Fatal(err)
	}
	parsedHTML, err := Parse(strings.NewReader(string(htmlExport)))
	if err != nil {
		t.Fatalf("exported HTML did not import: %v\n%s", err, htmlExport)
	}
	if len(parsedHTML.Categories) != 1 || parsedHTML.Categories[0].Name != "公开 & 工具" || parsedHTML.Categories[0].Links[0].Name != "Docs <One>" {
		t.Fatalf("HTML round trip lost data: %+v", parsedHTML)
	}
	jsonExport, err := service.ExportJSON(context.Background(), ScopeAll)
	if err != nil {
		t.Fatal(err)
	}
	parsedJSON, err := Parse(strings.NewReader(string(jsonExport)))
	if err != nil {
		t.Fatalf("exported JSON did not import: %v\n%s", err, jsonExport)
	}
	if len(parsedJSON.Categories) != 2 || parsedJSON.Categories[0].IconName != "tools" || parsedJSON.Categories[0].Links[0].Description != "Reference & notes" || parsedJSON.Categories[1].Visibility != navigation.VisibilityPrivate {
		t.Fatalf("JSON round trip lost data: %+v", parsedJSON)
	}
}

func TestExportRejectsUnknownScope(t *testing.T) {
	t.Parallel()
	_, err := New(&fakeRepository{}).ExportJSON(context.Background(), "unknown")
	if !errors.Is(err, ErrInvalidExportScope) {
		t.Fatalf("expected scope error, got %v", err)
	}
}

type fakeRepository struct {
	groups []navigation.Group
}

func (repository *fakeRepository) Navigation(context.Context, bool) ([]navigation.Group, error) {
	return repository.groups, nil
}
func (repository *fakeRepository) ImportBookmarks(context.Context, Batch, ImportOptions) (CommitResult, error) {
	return CommitResult{}, nil
}
