package sqlite

import (
	"context"
	"testing"

	"iiiu-nav/internal/bookmarks"
	"iiiu-nav/internal/navigation"
)

func TestBookmarkImportDuplicateStrategiesAndCategoryMatching(t *testing.T) {
	t.Parallel()
	for _, strategy := range []bookmarks.DuplicateStrategy{bookmarks.DuplicateSkip, bookmarks.DuplicateUpdate, bookmarks.DuplicateCreate} {
		t.Run(string(strategy), func(t *testing.T) {
			store := openTestStore(t)
			category := createTestCategory(t, store, "Caf\u00e9", "cafe", 10)
			original, err := store.CreateLink(context.Background(), navigation.LinkInput{CategoryID: category.ID, Name: "Old", URL: "https://example.com/", IconSource: navigation.IconSourceGenerated, IconValue: "O", SortOrder: 10})
			if err != nil {
				t.Fatal(err)
			}
			batch := bookmarks.Batch{Format: bookmarks.FormatHTML, Categories: []bookmarks.Category{{
				Name: "CAFE\u0301", Links: []bookmarks.Link{{Name: "New", URL: "HTTPS://EXAMPLE.COM:443#top", IconSource: navigation.IconSourceGenerated, IconValue: "N"}},
			}}}
			result, err := store.ImportBookmarks(context.Background(), batch, bookmarks.ImportOptions{Visibility: navigation.VisibilityPrivate, Duplicates: strategy})
			if err != nil {
				t.Fatal(err)
			}
			groups, err := store.Navigation(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			if result.CreatedCategories != 0 || len(groups) != 1 {
				t.Fatalf("category should be reused: result=%+v groups=%+v", result, groups)
			}
			switch strategy {
			case bookmarks.DuplicateSkip:
				if result.SkippedLinks != 1 || len(groups[0].Links) != 1 || groups[0].Links[0].Name != "Old" {
					t.Fatalf("unexpected skip result: %+v %+v", result, groups)
				}
			case bookmarks.DuplicateUpdate:
				if result.UpdatedLinks != 1 || len(groups[0].Links) != 1 || groups[0].Links[0].ID != original.ID || groups[0].Links[0].Name != "New" {
					t.Fatalf("unexpected update result: %+v %+v", result, groups)
				}
			case bookmarks.DuplicateCreate:
				if result.CreatedLinks != 1 || len(groups[0].Links) != 2 {
					t.Fatalf("unexpected create result: %+v %+v", result, groups)
				}
			}
		})
	}
}

func TestBookmarkImportIsTransactional(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	batch := bookmarks.Batch{Categories: []bookmarks.Category{{Name: "Import", Links: []bookmarks.Link{
		{Name: "Valid", URL: "https://valid.example", IconSource: navigation.IconSourceGenerated, IconValue: "V"},
		{Name: "Break constraint", URL: "https://invalid-source.example", IconSource: "unsupported", IconValue: "X"},
	}}}}
	_, err := store.ImportBookmarks(context.Background(), batch, bookmarks.ImportOptions{Visibility: navigation.VisibilityPrivate, Duplicates: bookmarks.DuplicateSkip})
	if err == nil {
		t.Fatal("expected import failure")
	}
	groups, readErr := store.Navigation(context.Background(), true)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(groups) != 0 {
		t.Fatalf("failed import left partial records: %+v", groups)
	}
}
