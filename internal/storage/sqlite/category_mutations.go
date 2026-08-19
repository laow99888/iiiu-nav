package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"iiiu-nav/internal/navigation"
)

func (store *Store) UpdateCategory(
	ctx context.Context,
	id int64,
	input navigation.CategoryInput,
) (navigation.Category, error) {
	now := time.Now().UTC()
	result, err := store.database.ExecContext(
		ctx,
		`UPDATE categories
		 SET name = ?, icon_name = ?, visibility = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name,
		input.IconName,
		input.Visibility,
		timestamp(now),
		id,
	)
	if err != nil {
		return navigation.Category{}, fmt.Errorf("update category: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return navigation.Category{}, fmt.Errorf("read updated category count: %w", err)
	}
	if affected == 0 {
		return navigation.Category{}, navigation.ErrCategoryNotFound
	}
	return store.categoryByID(ctx, id)
}

func (store *Store) DeleteCategory(
	ctx context.Context,
	id int64,
	mode navigation.CategoryDeleteMode,
	targetID int64,
) error {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin category deletion: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	var exists int
	if err := transaction.QueryRowContext(ctx, `SELECT 1 FROM categories WHERE id = ?`, id).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return navigation.ErrCategoryNotFound
	} else if err != nil {
		return fmt.Errorf("find deleted category: %w", err)
	}

	var linkCount int
	if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE category_id = ?`, id).Scan(&linkCount); err != nil {
		return fmt.Errorf("count category links: %w", err)
	}
	if linkCount > 0 {
		switch mode {
		case navigation.CategoryDeleteLinks:
			if _, err := transaction.ExecContext(ctx, `DELETE FROM links WHERE category_id = ?`, id); err != nil {
				return fmt.Errorf("delete category links: %w", err)
			}
		case navigation.CategoryMoveLinks:
			if targetID == 0 || targetID == id {
				return navigation.ErrInvalidCategoryMove
			}
			var targetExists int
			if err := transaction.QueryRowContext(ctx, `SELECT 1 FROM categories WHERE id = ?`, targetID).Scan(&targetExists); errors.Is(err, sql.ErrNoRows) {
				return navigation.ErrInvalidCategoryMove
			} else if err != nil {
				return fmt.Errorf("find category move target: %w", err)
			}
			var targetOffset int
			if err := transaction.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) FROM links WHERE category_id = ?`, targetID).Scan(&targetOffset); err != nil {
				return fmt.Errorf("read category move order: %w", err)
			}
			if _, err := transaction.ExecContext(
				ctx,
				`UPDATE links SET category_id = ?, sort_order = sort_order + ? WHERE category_id = ?`,
				targetID,
				targetOffset+10,
				id,
			); err != nil {
				return fmt.Errorf("move category links: %w", err)
			}
		default:
			return navigation.ErrCategoryHasLinks
		}
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit category deletion: %w", err)
	}
	return nil
}

func (store *Store) ReorderCategories(ctx context.Context, ids []int64) error {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin category reorder: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	var count int
	if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories`).Scan(&count); err != nil {
		return fmt.Errorf("count categories for reorder: %w", err)
	}
	if count != len(ids) {
		return navigation.ErrInvalidCategoryOrder
	}
	seen := make(map[int64]struct{}, len(ids))
	for index, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			return navigation.ErrInvalidCategoryOrder
		}
		seen[id] = struct{}{}
		result, err := transaction.ExecContext(
			ctx,
			`UPDATE categories SET sort_order = ?, updated_at = ? WHERE id = ?`,
			(index+1)*10,
			timestamp(time.Now().UTC()),
			id,
		)
		if err != nil {
			return fmt.Errorf("reorder category: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return navigation.ErrInvalidCategoryOrder
		}
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit category reorder: %w", err)
	}
	return nil
}

func (store *Store) categoryByID(ctx context.Context, id int64) (navigation.Category, error) {
	category, err := scanCategory(store.database.QueryRowContext(
		ctx,
		`SELECT id, name, slug, icon_name, visibility, sort_order, created_at, updated_at
		 FROM categories WHERE id = ?`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return navigation.Category{}, navigation.ErrCategoryNotFound
	}
	return category, err
}
