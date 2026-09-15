package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"iiiu-nav/internal/navigation"
)

func (store *Store) Navigation(ctx context.Context, includePrivate bool) ([]navigation.Group, error) {
	visibilityClause := `WHERE categories.visibility = 'public'`
	if includePrivate {
		visibilityClause = ""
	}
	rows, err := store.database.QueryContext(
		ctx,
		`SELECT
			categories.id, categories.name, categories.slug, categories.icon_name,
			categories.visibility, categories.sort_order, categories.created_at, categories.updated_at,
			links.id, links.category_id, links.name, links.description, links.url,
			links.icon_source, links.icon_value, links.sort_order, links.created_at, links.updated_at
		 FROM categories
		 LEFT JOIN links ON links.category_id = categories.id
		 `+visibilityClause+`
		 ORDER BY categories.sort_order, categories.id, links.sort_order, links.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("read navigation: %w", err)
	}
	defer rows.Close()

	groups := make([]navigation.Group, 0)
	var currentCategoryID int64
	for rows.Next() {
		category, link, hasLink, err := scanNavigationRow(rows)
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 || currentCategoryID != category.ID {
			groups = append(groups, navigation.Group{Category: category, Links: make([]navigation.Link, 0)})
			currentCategoryID = category.ID
		}
		if hasLink {
			groups[len(groups)-1].Links = append(groups[len(groups)-1].Links, link)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate navigation: %w", err)
	}
	return groups, nil
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func (store *Store) CreateCategory(ctx context.Context, input navigation.CategoryInput) (navigation.Category, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return navigation.Category{}, fmt.Errorf("begin category create: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()
	order := input.SortOrder
	if order <= 0 {
		if err := transaction.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) + 10 FROM categories`).Scan(&order); err != nil {
			return navigation.Category{}, fmt.Errorf("read created category order: %w", err)
		}
	}
	now := time.Now().UTC()
	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO categories
            (name, slug, icon_name, visibility, sort_order, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.Name,
		input.Slug,
		input.IconName,
		input.Visibility,
		order,
		timestamp(now),
		timestamp(now),
	)
	if err != nil {
		return navigation.Category{}, fmt.Errorf("create category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return navigation.Category{}, fmt.Errorf("read created category id: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return navigation.Category{}, fmt.Errorf("commit category create: %w", err)
	}
	return navigation.Category{
		ID:         id,
		Name:       input.Name,
		Slug:       input.Slug,
		IconName:   input.IconName,
		Visibility: input.Visibility,
		SortOrder:  order,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (store *Store) Categories(ctx context.Context) ([]navigation.Category, error) {
	rows, err := store.database.QueryContext(
		ctx,
		`SELECT id, name, slug, icon_name, visibility, sort_order, created_at, updated_at
         FROM categories
         ORDER BY sort_order, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := make([]navigation.Category, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

func (store *Store) CreateLink(ctx context.Context, input navigation.LinkInput) (navigation.Link, error) {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return navigation.Link{}, fmt.Errorf("begin link create: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var categoryCount int
	if err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE id = ?`, input.CategoryID).Scan(&categoryCount); err != nil {
		return navigation.Link{}, fmt.Errorf("check link category: %w", err)
	}
	if categoryCount == 0 {
		return navigation.Link{}, navigation.ErrCategoryNotFound
	}
	order := input.SortOrder
	if order <= 0 {
		if err := transaction.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) + 10 FROM links WHERE category_id = ?`, input.CategoryID).Scan(&order); err != nil {
			return navigation.Link{}, fmt.Errorf("read created link order: %w", err)
		}
	}
	now := time.Now().UTC()
	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO links
            (category_id, name, description, url, icon_source, icon_value, sort_order, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.CategoryID,
		input.Name,
		input.Description,
		input.URL,
		input.IconSource,
		input.IconValue,
		order,
		timestamp(now),
		timestamp(now),
	)
	if err != nil {
		return navigation.Link{}, fmt.Errorf("create link: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return navigation.Link{}, fmt.Errorf("read created link id: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return navigation.Link{}, fmt.Errorf("commit link create: %w", err)
	}
	return navigation.Link{
		ID:          id,
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		URL:         input.URL,
		IconSource:  input.IconSource,
		IconValue:   input.IconValue,
		SortOrder:   order,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (store *Store) LinksByCategory(ctx context.Context, categoryID int64) ([]navigation.Link, error) {
	rows, err := store.database.QueryContext(
		ctx,
		`SELECT id, category_id, name, description, url, icon_source, icon_value, sort_order, created_at, updated_at
         FROM links
         WHERE category_id = ?
         ORDER BY sort_order, id`,
		categoryID,
	)
	if err != nil {
		return nil, fmt.Errorf("list category links: %w", err)
	}
	defer rows.Close()

	links := make([]navigation.Link, 0)
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category links: %w", err)
	}
	return links, nil
}

func scanCategory(scanner rowScanner) (navigation.Category, error) {
	var category navigation.Category
	var createdAt int64
	var updatedAt int64
	if err := scanner.Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.IconName,
		&category.Visibility,
		&category.SortOrder,
		&createdAt,
		&updatedAt,
	); err != nil {
		return navigation.Category{}, fmt.Errorf("scan category: %w", err)
	}
	category.CreatedAt = timeFromTimestamp(createdAt)
	category.UpdatedAt = timeFromTimestamp(updatedAt)
	return category, nil
}

func scanLink(scanner rowScanner) (navigation.Link, error) {
	var link navigation.Link
	var createdAt int64
	var updatedAt int64
	if err := scanner.Scan(
		&link.ID,
		&link.CategoryID,
		&link.Name,
		&link.Description,
		&link.URL,
		&link.IconSource,
		&link.IconValue,
		&link.SortOrder,
		&createdAt,
		&updatedAt,
	); err != nil {
		return navigation.Link{}, fmt.Errorf("scan link: %w", err)
	}
	link.CreatedAt = timeFromTimestamp(createdAt)
	link.UpdatedAt = timeFromTimestamp(updatedAt)
	return link, nil
}

func scanNavigationRow(scanner rowScanner) (navigation.Category, navigation.Link, bool, error) {
	var category navigation.Category
	var categoryCreatedAt int64
	var categoryUpdatedAt int64
	var linkID sql.NullInt64
	var linkCategoryID sql.NullInt64
	var linkName sql.NullString
	var linkDescription sql.NullString
	var linkURL sql.NullString
	var linkIconSource sql.NullString
	var linkIconValue sql.NullString
	var linkSortOrder sql.NullInt64
	var linkCreatedAt sql.NullInt64
	var linkUpdatedAt sql.NullInt64
	if err := scanner.Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.IconName,
		&category.Visibility,
		&category.SortOrder,
		&categoryCreatedAt,
		&categoryUpdatedAt,
		&linkID,
		&linkCategoryID,
		&linkName,
		&linkDescription,
		&linkURL,
		&linkIconSource,
		&linkIconValue,
		&linkSortOrder,
		&linkCreatedAt,
		&linkUpdatedAt,
	); err != nil {
		return navigation.Category{}, navigation.Link{}, false, fmt.Errorf("scan navigation: %w", err)
	}
	category.CreatedAt = timeFromTimestamp(categoryCreatedAt)
	category.UpdatedAt = timeFromTimestamp(categoryUpdatedAt)
	if !linkID.Valid {
		return category, navigation.Link{}, false, nil
	}
	link := navigation.Link{
		ID:          linkID.Int64,
		CategoryID:  linkCategoryID.Int64,
		Name:        linkName.String,
		Description: linkDescription.String,
		URL:         linkURL.String,
		IconSource:  navigation.IconSource(linkIconSource.String),
		IconValue:   linkIconValue.String,
		SortOrder:   int(linkSortOrder.Int64),
		CreatedAt:   timeFromTimestamp(linkCreatedAt.Int64),
		UpdatedAt:   timeFromTimestamp(linkUpdatedAt.Int64),
	}
	return category, link, true, nil
}

var _ rowScanner = (*sql.Row)(nil)
var _ rowScanner = (*sql.Rows)(nil)
