package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"iiiu-nav/internal/navigation"
)

func (store *Store) UpdateLink(ctx context.Context, id int64, input navigation.LinkInput) (navigation.Link, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return navigation.Link{}, fmt.Errorf("begin link update: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var currentCategoryID int64
	var currentOrder int
	if err := transaction.QueryRowContext(ctx, `SELECT category_id, sort_order FROM links WHERE id = ?`, id).Scan(&currentCategoryID, &currentOrder); errors.Is(err, sql.ErrNoRows) {
		return navigation.Link{}, navigation.ErrLinkNotFound
	} else if err != nil {
		return navigation.Link{}, fmt.Errorf("find updated link: %w", err)
	}
	order := currentOrder
	if currentCategoryID != input.CategoryID {
		var categoryCount int
		if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE id = ?`, input.CategoryID).Scan(&categoryCount); err != nil {
			return navigation.Link{}, fmt.Errorf("check link category: %w", err)
		}
		if categoryCount == 0 {
			return navigation.Link{}, navigation.ErrCategoryNotFound
		}
		if err := transaction.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) + 10 FROM links WHERE category_id = ?`, input.CategoryID).Scan(&order); err != nil {
			return navigation.Link{}, fmt.Errorf("read target link order: %w", err)
		}
	}
	result, err := transaction.ExecContext(ctx, `UPDATE links SET category_id = ?, name = ?, description = ?, url = ?, icon_source = ?, icon_value = ?, sort_order = ?, updated_at = ? WHERE id = ?`,
		input.CategoryID, input.Name, input.Description, input.URL, input.IconSource, input.IconValue, order, timestamp(time.Now().UTC()), id)
	if err != nil {
		return navigation.Link{}, fmt.Errorf("update link: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		return navigation.Link{}, navigation.ErrLinkNotFound
	}
	if err := transaction.Commit(); err != nil {
		return navigation.Link{}, fmt.Errorf("commit link update: %w", err)
	}
	return store.linkByID(ctx, id)
}

func (store *Store) DeleteLink(ctx context.Context, id int64) error {
	result, err := store.database.ExecContext(ctx, `DELETE FROM links WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted link count: %w", err)
	}
	if affected == 0 {
		return navigation.ErrLinkNotFound
	}
	return nil
}

func (store *Store) Link(ctx context.Context, id int64) (navigation.Link, error) {
	return store.linkByID(ctx, id)
}

func (store *Store) Links(ctx context.Context) ([]navigation.Link, error) {
	rows, err := store.database.QueryContext(ctx, `SELECT id, category_id, name, description, url, icon_source, icon_value, sort_order, created_at, updated_at FROM links ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()
	links := make([]navigation.Link, 0)
	for rows.Next() {
		link, scanErr := scanLink(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate links: %w", err)
	}
	return links, nil
}

func (store *Store) LogoReferenceCount(ctx context.Context, publicPath string) (int, error) {
	var count int
	if err := store.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE icon_value = ?`, publicPath).Scan(&count); err != nil {
		return 0, fmt.Errorf("count logo references: %w", err)
	}
	return count, nil
}

func (store *Store) ReorderLinks(ctx context.Context, categoryID int64, ids []int64) error {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin link reorder: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var count int
	if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE category_id = ?`, categoryID).Scan(&count); err != nil {
		return fmt.Errorf("count links for reorder: %w", err)
	}
	if count != len(ids) {
		return navigation.ErrInvalidLinkOrder
	}
	seen := make(map[int64]struct{}, len(ids))
	for index, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			return navigation.ErrInvalidLinkOrder
		}
		seen[id] = struct{}{}
		result, err := transaction.ExecContext(ctx, `UPDATE links SET sort_order = ?, updated_at = ? WHERE id = ? AND category_id = ?`, (index+1)*10, timestamp(time.Now().UTC()), id, categoryID)
		if err != nil {
			return fmt.Errorf("reorder link: %w", err)
		}
		if affected, err := result.RowsAffected(); err != nil || affected != 1 {
			return navigation.ErrInvalidLinkOrder
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit link reorder: %w", err)
	}
	return nil
}

func (store *Store) linkByID(ctx context.Context, id int64) (navigation.Link, error) {
	link, err := scanLink(store.database.QueryRowContext(ctx, `SELECT id, category_id, name, description, url, icon_source, icon_value, sort_order, created_at, updated_at FROM links WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return navigation.Link{}, navigation.ErrLinkNotFound
	}
	return link, err
}
