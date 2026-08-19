package sqlite

import (
	"context"
	"errors"
	"testing"

	"iiiu-nav/internal/navigation"
)

func TestLinkMutationsMoveReorderAndDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	firstCategory := createTestCategory(t, store, "First", "first-links", 10)
	secondCategory := createTestCategory(t, store, "Second", "second-links", 20)
	first := createTestLink(t, store, firstCategory.ID, "First", 10)
	second := createTestLink(t, store, firstCategory.ID, "Second", 20)

	updated, err := store.UpdateLink(ctx, first.ID, navigation.LinkInput{
		CategoryID: secondCategory.ID, Name: "Moved", Description: "Updated",
		URL: "https://moved.example/", IconSource: navigation.IconSourceGenerated, IconValue: "M",
	})
	if err != nil || updated.CategoryID != secondCategory.ID || updated.Name != "Moved" {
		t.Fatalf("unexpected moved link: link=%+v err=%v", updated, err)
	}
	if links, err := store.LinksByCategory(ctx, firstCategory.ID); err != nil || len(links) != 1 || links[0].ID != second.ID {
		t.Fatalf("unexpected source links: links=%+v err=%v", links, err)
	}

	third := createTestLink(t, store, secondCategory.ID, "Third", 20)
	if err := store.ReorderLinks(ctx, secondCategory.ID, []int64{third.ID, first.ID}); err != nil {
		t.Fatalf("reorder links: %v", err)
	}
	ordered, err := store.LinksByCategory(ctx, secondCategory.ID)
	if err != nil || ordered[0].ID != third.ID || ordered[1].ID != first.ID {
		t.Fatalf("unexpected ordered links: links=%+v err=%v", ordered, err)
	}
	if err := store.ReorderLinks(ctx, secondCategory.ID, []int64{first.ID}); !errors.Is(err, navigation.ErrInvalidLinkOrder) {
		t.Fatalf("expected stale order error, got %v", err)
	}
	if err := store.DeleteLink(ctx, first.ID); err != nil {
		t.Fatalf("delete link: %v", err)
	}
	if err := store.DeleteLink(ctx, first.ID); !errors.Is(err, navigation.ErrLinkNotFound) {
		t.Fatalf("expected missing link error, got %v", err)
	}
}

func TestLinkMoveToMissingCategoryRollsBack(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	category := createTestCategory(t, store, "Source", "rollback-links", 10)
	link := createTestLink(t, store, category.ID, "Stable", 10)
	_, err := store.UpdateLink(ctx, link.ID, navigation.LinkInput{
		CategoryID: 9999, Name: "Changed", URL: "https://changed.example/",
		IconSource: navigation.IconSourceGenerated, IconValue: "C",
	})
	if err == nil {
		t.Fatal("expected missing target category to fail")
	}
	links, err := store.LinksByCategory(ctx, category.ID)
	if err != nil || len(links) != 1 || links[0].Name != "Stable" {
		t.Fatalf("failed move changed link: links=%+v err=%v", links, err)
	}
}

func createTestLink(t *testing.T, store *Store, categoryID int64, name string, order int) navigation.Link {
	t.Helper()
	link, err := store.CreateLink(context.Background(), navigation.LinkInput{
		CategoryID: categoryID, Name: name, URL: "https://" + name + ".example/",
		IconSource: navigation.IconSourceGenerated, IconValue: name[:1], SortOrder: order,
	})
	if err != nil {
		t.Fatalf("create link %q: %v", name, err)
	}
	return link
}
