package navigation

import (
	"errors"
	"time"
)

type Visibility string

type IconSource string

type CategoryDeleteMode string

const (
	VisibilityPublic    Visibility         = "public"
	VisibilityPrivate   Visibility         = "private"
	IconSourceAuto      IconSource         = "auto"
	IconSourceURL       IconSource         = "url"
	IconSourceUpload    IconSource         = "upload"
	IconSourceGenerated IconSource         = "generated"
	CategoryDeleteLinks CategoryDeleteMode = "delete"
	CategoryMoveLinks   CategoryDeleteMode = "move"
)

var (
	ErrCategoryHasLinks     = errors.New("category has links")
	ErrCategoryNotFound     = errors.New("category not found")
	ErrInvalidCategoryMove  = errors.New("invalid category move target")
	ErrInvalidCategoryOrder = errors.New("category order must contain every category")
	ErrLinkNotFound         = errors.New("link not found")
	ErrInvalidLinkOrder     = errors.New("link order must contain every category link")
)

type Category struct {
	ID         int64
	Name       string
	Slug       string
	IconName   string
	Visibility Visibility
	SortOrder  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CategoryInput struct {
	Name       string
	Slug       string
	IconName   string
	Visibility Visibility
	SortOrder  int
}

type Link struct {
	ID          int64
	CategoryID  int64
	Name        string
	Description string
	URL         string
	IconSource  IconSource
	IconValue   string
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type LinkInput struct {
	CategoryID  int64
	Name        string
	Description string
	URL         string
	IconSource  IconSource
	IconValue   string
	SortOrder   int
}

type Group struct {
	Category Category
	Links    []Link
}
