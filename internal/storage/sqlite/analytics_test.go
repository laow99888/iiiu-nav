package sqlite

import (
	"context"
	"testing"
)

func TestPageViewsIncrementAndRetention(t *testing.T) {
	t.Parallel()
	store := openTestStore(t)
	ctx := context.Background()

	if err := store.IncrementPageViews(ctx, "2025-08-19", "2025-01-01"); err != nil {
		t.Fatalf("seed expired page views: %v", err)
	}
	if err := store.IncrementPageViews(ctx, "2026-08-19", "2025-08-20"); err != nil {
		t.Fatalf("record first current page view: %v", err)
	}
	if err := store.IncrementPageViews(ctx, "2026-08-19", "2025-08-20"); err != nil {
		t.Fatalf("record second current page view: %v", err)
	}

	series, err := store.PageViews(ctx, "2025-01-01")
	if err != nil {
		t.Fatalf("read page views: %v", err)
	}
	if len(series) != 1 || series[0].Day != "2026-08-19" || series[0].Views != 2 {
		t.Fatalf("unexpected retained page views: %+v", series)
	}
}
