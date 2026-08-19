package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"iiiu-nav/internal/navigation"
)

func TestNavigationRepositoryPersistsOrderedRecords(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)

	second, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name:       "开发",
		Slug:       "development",
		Visibility: navigation.VisibilityPrivate,
		SortOrder:  20,
	})
	if err != nil {
		t.Fatalf("create second category: %v", err)
	}
	first, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name:       "常用",
		Slug:       "frequent",
		IconName:   "star",
		Visibility: navigation.VisibilityPublic,
		SortOrder:  10,
	})
	if err != nil {
		t.Fatalf("create first category: %v", err)
	}

	categories, err := store.Categories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 2 || categories[0].ID != first.ID || categories[1].ID != second.ID {
		t.Fatalf("unexpected category order: %+v", categories)
	}

	link, err := store.CreateLink(ctx, navigation.LinkInput{
		CategoryID:  first.ID,
		Name:        "Go",
		Description: "Go language",
		URL:         "https://go.dev/",
		IconSource:  navigation.IconSourceAuto,
		SortOrder:   5,
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	links, err := store.LinksByCategory(ctx, first.ID)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 || links[0].ID != link.ID || links[0].URL != "https://go.dev/" {
		t.Fatalf("unexpected links: %+v", links)
	}
	if links[0].CreatedAt.IsZero() || links[0].UpdatedAt.IsZero() {
		t.Fatal("expected persisted timestamps")
	}
}

func TestNavigationRepositoryPersistsAcrossReopen(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "persistent.db")
	firstStore, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	category, err := firstStore.CreateCategory(ctx, navigation.CategoryInput{
		Name:       "Persistent",
		Slug:       "persistent",
		Visibility: navigation.VisibilityPublic,
	})
	if err != nil {
		_ = firstStore.Close()
		t.Fatalf("create category: %v", err)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	secondStore, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() {
		if err := secondStore.Close(); err != nil {
			t.Errorf("close second store: %v", err)
		}
	})

	categories, err := secondStore.Categories(ctx)
	if err != nil {
		t.Fatalf("list categories after reopen: %v", err)
	}
	if len(categories) != 1 || categories[0].ID != category.ID {
		t.Fatalf("unexpected categories after reopen: %+v", categories)
	}
}

func TestNavigationRepositoryEnforcesDatabaseConstraints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)

	if _, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name:       "Invalid",
		Slug:       "invalid",
		Visibility: navigation.Visibility("secret"),
	}); err == nil {
		t.Fatal("expected invalid visibility to fail")
	}

	if _, err := store.CreateLink(ctx, navigation.LinkInput{
		CategoryID: 9999,
		Name:       "Missing category",
		URL:        "https://example.com/",
		IconSource: navigation.IconSourceGenerated,
	}); err == nil {
		t.Fatal("expected missing category to fail")
	}
}

func TestNavigationFiltersPrivateGroupsAndKeepsEmptyCategories(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	publicEmpty, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name: "Empty", Slug: "empty", Visibility: navigation.VisibilityPublic, SortOrder: 10,
	})
	if err != nil {
		t.Fatalf("create empty category: %v", err)
	}
	public, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name: "Public", Slug: "public", Visibility: navigation.VisibilityPublic, SortOrder: 20,
	})
	if err != nil {
		t.Fatalf("create public category: %v", err)
	}
	private, err := store.CreateCategory(ctx, navigation.CategoryInput{
		Name: "Private", Slug: "private", Visibility: navigation.VisibilityPrivate, SortOrder: 30,
	})
	if err != nil {
		t.Fatalf("create private category: %v", err)
	}
	for _, input := range []navigation.LinkInput{
		{CategoryID: public.ID, Name: "Second", URL: "https://second.example", IconSource: navigation.IconSourceGenerated, SortOrder: 20},
		{CategoryID: public.ID, Name: "First", URL: "https://first.example", IconSource: navigation.IconSourceGenerated, SortOrder: 10},
		{CategoryID: private.ID, Name: "Secret", URL: "https://secret.example", IconSource: navigation.IconSourceGenerated},
	} {
		if _, err := store.CreateLink(ctx, input); err != nil {
			t.Fatalf("create link %q: %v", input.Name, err)
		}
	}

	publicGroups, err := store.Navigation(ctx, false)
	if err != nil {
		t.Fatalf("read public navigation: %v", err)
	}
	if len(publicGroups) != 2 || publicGroups[0].Category.ID != publicEmpty.ID || len(publicGroups[0].Links) != 0 {
		t.Fatalf("unexpected public groups: %+v", publicGroups)
	}
	if got := publicGroups[1].Links; len(got) != 2 || got[0].Name != "First" || got[1].Name != "Second" {
		t.Fatalf("unexpected public link order: %+v", got)
	}

	adminGroups, err := store.Navigation(ctx, true)
	if err != nil {
		t.Fatalf("read administrator navigation: %v", err)
	}
	if len(adminGroups) != 3 || adminGroups[2].Category.ID != private.ID || adminGroups[2].Links[0].Name != "Secret" {
		t.Fatalf("unexpected administrator groups: %+v", adminGroups)
	}
}

func TestNavigationReadsTwoThousandLinksWithIndexedJoin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	seedNavigationLinks(t, store, 2_000)

	started := time.Now()
	groups, err := store.Navigation(ctx, false)
	if err != nil {
		t.Fatalf("read large navigation: %v", err)
	}
	links := 0
	for _, group := range groups {
		links += len(group.Links)
	}
	if links != 2_000 {
		t.Fatalf("expected 2,000 links, got %d", links)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("2,000-link navigation read took %s", elapsed)
	}

	rows, err := store.database.QueryContext(ctx, `EXPLAIN QUERY PLAN
		SELECT links.id FROM categories
		LEFT JOIN links ON links.category_id = categories.id
		WHERE categories.visibility = 'public'
		ORDER BY categories.sort_order, categories.id, links.sort_order, links.id`)
	if err != nil {
		t.Fatalf("explain navigation query: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatalf("scan navigation query plan: %v", err)
		}
		plan.WriteString(detail)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate navigation query plan: %v", err)
	}
	if !strings.Contains(plan.String(), "links_category_order_idx") {
		t.Fatalf("navigation join did not use category/order index:\n%s", plan.String())
	}
}

func BenchmarkNavigationTwoThousandLinks(b *testing.B) {
	store := openTestStore(b)
	seedNavigationLinks(b, store, 2_000)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		groups, err := store.Navigation(context.Background(), false)
		if err != nil || len(groups) == 0 {
			b.Fatalf("read navigation: groups=%d err=%v", len(groups), err)
		}
	}
}

type testingTB interface {
	Helper()
	Fatalf(string, ...any)
}

func seedNavigationLinks(tb testingTB, store *Store, count int) {
	tb.Helper()
	ctx := context.Background()
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		tb.Fatalf("begin navigation seed: %v", err)
	}
	defer transaction.Rollback()
	now := timestamp(time.Now().UTC())
	categoryIDs := make([]int64, 20)
	for index := range categoryIDs {
		result, err := transaction.ExecContext(ctx, `INSERT INTO categories
			(name, slug, icon_name, visibility, sort_order, created_at, updated_at)
			VALUES (?, ?, 'folder', 'public', ?, ?, ?)`,
			fmt.Sprintf("Category %02d", index+1), fmt.Sprintf("category-%02d", index+1), (index+1)*10, now, now)
		if err != nil {
			tb.Fatalf("seed category %d: %v", index, err)
		}
		categoryIDs[index], err = result.LastInsertId()
		if err != nil {
			tb.Fatalf("read seeded category id: %v", err)
		}
	}
	for index := range count {
		_, err := transaction.ExecContext(ctx, `INSERT INTO links
			(category_id, name, description, url, icon_source, icon_value, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'generated', 'L', ?, ?, ?)`,
			categoryIDs[index%len(categoryIDs)], fmt.Sprintf("Performance Link %04d", index+1),
			"Representative navigation description", fmt.Sprintf("https://example.com/item/%d", index+1),
			(index/len(categoryIDs)+1)*10, now, now)
		if err != nil {
			tb.Fatalf("seed link %d: %v", index, err)
		}
	}
	if err := transaction.Commit(); err != nil {
		tb.Fatalf("commit navigation seed: %v", err)
	}
}
