package sqlite

import (
	"context"
	"fmt"

	"iiiu-nav/internal/analytics"
)

func (store *Store) IncrementPageViews(ctx context.Context, day, retentionCutoff string) error {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin page-view update: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, `DELETE FROM daily_page_views WHERE day < ?`, retentionCutoff); err != nil {
		return fmt.Errorf("remove expired page views: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO daily_page_views (day, views) VALUES (?, 1)
		ON CONFLICT(day) DO UPDATE SET views = views + 1
	`, day); err != nil {
		return fmt.Errorf("increment daily page views: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit page-view update: %w", err)
	}
	return nil
}

func (store *Store) PageViews(ctx context.Context, since string) ([]analytics.DailyPageViews, error) {
	rows, err := store.database.QueryContext(ctx, `
		SELECT day, views
		FROM daily_page_views
		WHERE day >= ?
		ORDER BY day
	`, since)
	if err != nil {
		return nil, fmt.Errorf("read daily page views: %w", err)
	}
	defer rows.Close()
	result := make([]analytics.DailyPageViews, 0)
	for rows.Next() {
		var item analytics.DailyPageViews
		if err := rows.Scan(&item.Day, &item.Views); err != nil {
			return nil, fmt.Errorf("scan daily page views: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily page views: %w", err)
	}
	return result, nil
}
