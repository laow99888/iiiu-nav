package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
	"unicode/utf8"

	"iiiu-nav/internal/bookmarks"
)

func (store *Store) ImportBookmarks(ctx context.Context, batch bookmarks.Batch, options bookmarks.ImportOptions) (bookmarks.CommitResult, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return bookmarks.CommitResult{}, fmt.Errorf("begin bookmark import: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	categories, nextCategoryOrder, err := importCategories(ctx, transaction)
	if err != nil {
		return bookmarks.CommitResult{}, err
	}
	links, nextLinkOrder, err := importLinks(ctx, transaction)
	if err != nil {
		return bookmarks.CommitResult{}, err
	}
	result := bookmarks.CommitResult{ImportedLinkIDs: make([]int64, 0)}
	for _, category := range batch.Categories {
		if category.Name == "" || utf8.RuneCountInString(category.Name) > 80 {
			result.InvalidLinks += len(category.Links)
			continue
		}
		categoryID, exists := categories[bookmarks.CategoryKey(category.Name)]
		if !exists {
			slug, slugErr := importCategorySlug()
			if slugErr != nil {
				return bookmarks.CommitResult{}, fmt.Errorf("create import category slug: %w", slugErr)
			}
			nextCategoryOrder += 10
			created, createErr := transaction.ExecContext(ctx,
				`INSERT INTO categories (name, slug, icon_name, visibility, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				category.Name, slug, category.IconName, options.Visibility, nextCategoryOrder, timestampNow(), timestampNow())
			if createErr != nil {
				return bookmarks.CommitResult{}, fmt.Errorf("create imported category: %w", createErr)
			}
			categoryID, err = created.LastInsertId()
			if err != nil {
				return bookmarks.CommitResult{}, fmt.Errorf("read imported category id: %w", err)
			}
			categories[bookmarks.CategoryKey(category.Name)] = categoryID
			result.CreatedCategories++
		}
		for _, link := range category.Links {
			if link.Problem != "" {
				result.InvalidLinks++
				continue
			}
			canonical := bookmarks.CanonicalURL(link.URL)
			existingID, duplicate := links[canonical]
			if duplicate && options.Duplicates == bookmarks.DuplicateSkip {
				result.SkippedLinks++
				continue
			}
			if duplicate && options.Duplicates == bookmarks.DuplicateUpdate {
				nextLinkOrder[categoryID] += 10
				_, err = transaction.ExecContext(ctx,
					`UPDATE links SET category_id = ?, name = ?, description = ?, url = ?, icon_source = ?, icon_value = ?, sort_order = ?, updated_at = ? WHERE id = ?`,
					categoryID, link.Name, link.Description, link.URL, link.IconSource, link.IconValue, nextLinkOrder[categoryID], timestampNow(), existingID)
				if err != nil {
					return bookmarks.CommitResult{}, fmt.Errorf("update duplicate bookmark: %w", err)
				}
				result.UpdatedLinks++
				result.ImportedLinkIDs = append(result.ImportedLinkIDs, existingID)
				continue
			}
			nextLinkOrder[categoryID] += 10
			created, createErr := transaction.ExecContext(ctx,
				`INSERT INTO links (category_id, name, description, url, icon_source, icon_value, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				categoryID, link.Name, link.Description, link.URL, link.IconSource, link.IconValue, nextLinkOrder[categoryID], timestampNow(), timestampNow())
			if createErr != nil {
				return bookmarks.CommitResult{}, fmt.Errorf("create imported bookmark: %w", createErr)
			}
			linkID, idErr := created.LastInsertId()
			if idErr != nil {
				return bookmarks.CommitResult{}, fmt.Errorf("read imported bookmark id: %w", idErr)
			}
			links[canonical] = linkID
			result.CreatedLinks++
			result.ImportedLinkIDs = append(result.ImportedLinkIDs, linkID)
		}
	}
	if err := transaction.Commit(); err != nil {
		return bookmarks.CommitResult{}, fmt.Errorf("commit bookmark import: %w", err)
	}
	return result, nil
}

func importCategories(ctx context.Context, transaction *sql.Tx) (map[string]int64, int, error) {
	rows, err := transaction.QueryContext(ctx, `SELECT id, name, sort_order FROM categories`)
	if err != nil {
		return nil, 0, fmt.Errorf("read import categories: %w", err)
	}
	defer rows.Close()
	result := make(map[string]int64)
	maximumOrder := 0
	for rows.Next() {
		var id int64
		var name string
		var order int
		if err := rows.Scan(&id, &name, &order); err != nil {
			return nil, 0, fmt.Errorf("scan import category: %w", err)
		}
		result[bookmarks.CategoryKey(name)] = id
		if order > maximumOrder {
			maximumOrder = order
		}
	}
	return result, maximumOrder, rows.Err()
}

func importLinks(ctx context.Context, transaction *sql.Tx) (map[string]int64, map[int64]int, error) {
	rows, err := transaction.QueryContext(ctx, `SELECT id, category_id, url, sort_order FROM links ORDER BY id`)
	if err != nil {
		return nil, nil, fmt.Errorf("read import links: %w", err)
	}
	defer rows.Close()
	links := make(map[string]int64)
	orders := make(map[int64]int)
	for rows.Next() {
		var id, categoryID int64
		var rawURL string
		var order int
		if err := rows.Scan(&id, &categoryID, &rawURL, &order); err != nil {
			return nil, nil, fmt.Errorf("scan import link: %w", err)
		}
		if _, exists := links[bookmarks.CanonicalURL(rawURL)]; !exists {
			links[bookmarks.CanonicalURL(rawURL)] = id
		}
		if order > orders[categoryID] {
			orders[categoryID] = order
		}
	}
	return links, orders, rows.Err()
}

func importCategorySlug() (string, error) {
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "category-" + hex.EncodeToString(value), nil
}

func timestampNow() int64 { return timestamp(time.Now().UTC()) }
