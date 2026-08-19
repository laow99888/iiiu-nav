package sqlite

import (
	"context"
	"errors"
	"testing"

	"iiiu-nav/internal/navigation"
)

func TestCategoryMutationsPersistUpdateAndOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	first := createTestCategory(t, store, "First", "first", 10)
	second := createTestCategory(t, store, "Second", "second", 20)

	updated, err := store.UpdateCategory(ctx, first.ID, navigation.CategoryInput{
		Name: "Renamed", IconName: "code", Visibility: navigation.VisibilityPrivate,
	})
	if err != nil || updated.Name != "Renamed" || updated.IconName != "code" || updated.Visibility != navigation.VisibilityPrivate {
		t.Fatalf("unexpected update: category=%+v err=%v", updated, err)
	}
	if err := store.ReorderCategories(ctx, []int64{second.ID, first.ID}); err != nil {
		t.Fatalf("reorder categories: %v", err)
	}
	categories, err := store.Categories(ctx)
	if err != nil || categories[0].ID != second.ID || categories[1].ID != first.ID {
		t.Fatalf("unexpected persisted order: categories=%+v err=%v", categories, err)
	}
	if err := store.ReorderCategories(ctx, []int64{first.ID}); !errors.Is(err, navigation.ErrInvalidCategoryOrder) {
		t.Fatalf("expected stale order error, got %v", err)
	}
}

func TestCategoryDeletionMovesOrDeletesLinksTransactionally(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openTestStore(t)
	source := createTestCategory(t, store, "Source", "source", 10)
	target := createTestCategory(t, store, "Target", "target", 20)
	link, err := store.CreateLink(ctx, navigation.LinkInput{
		CategoryID: source.ID, Name: "Moved", URL: "https://example.com", IconSource: navigation.IconSourceGenerated,
	})
	if err != nil {
		t.Fatalf("create source link: %v", err)
	}

	if err := store.DeleteCategory(ctx, source.ID, navigation.CategoryMoveLinks, 9999); !errors.Is(err, navigation.ErrInvalidCategoryMove) {
		t.Fatalf("expected invalid target error, got %v", err)
	}
	if links, err := store.LinksByCategory(ctx, source.ID); err != nil || len(links) != 1 {
		t.Fatalf("failed move changed active data: links=%+v err=%v", links, err)
	}
	if err := store.DeleteCategory(ctx, source.ID, navigation.CategoryMoveLinks, target.ID); err != nil {
		t.Fatalf("move links and delete category: %v", err)
	}
	moved, err := store.LinksByCategory(ctx, target.ID)
	if err != nil || len(moved) != 1 || moved[0].ID != link.ID {
		t.Fatalf("unexpected moved links: links=%+v err=%v", moved, err)
	}

	deleteSource := createTestCategory(t, store, "Delete", "delete", 30)
	if _, err := store.CreateLink(ctx, navigation.LinkInput{
		CategoryID: deleteSource.ID, Name: "Deleted", URL: "https://delete.example", IconSource: navigation.IconSourceGenerated,
	}); err != nil {
		t.Fatalf("create deleted link: %v", err)
	}
	if err := store.DeleteCategory(ctx, deleteSource.ID, navigation.CategoryDeleteLinks, 0); err != nil {
		t.Fatalf("delete category links: %v", err)
	}
	if links, err := store.LinksByCategory(ctx, deleteSource.ID); err != nil || len(links) != 0 {
		t.Fatalf("deleted links remain: links=%+v err=%v", links, err)
	}
}

func createTestCategory(t *testing.T, store *Store, name, slug string, order int) navigation.Category {
	t.Helper()
	category, err := store.CreateCategory(context.Background(), navigation.CategoryInput{
		Name: name, Slug: slug, Visibility: navigation.VisibilityPublic, SortOrder: order,
	})
	if err != nil {
		t.Fatalf("create category %q: %v", name, err)
	}
	return category
}
