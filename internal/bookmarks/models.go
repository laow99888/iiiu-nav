package bookmarks

import (
	"context"

	"iiiu-nav/internal/navigation"
)

const (
	MaxImportBytes = 20 << 20
	UnfiledName    = "未分类"
)

type Format string
type DuplicateStrategy string
type EntryStatus string
type CategoryMapping string

const (
	FormatHTML Format = "html"
	FormatJSON Format = "json"

	DuplicateSkip   DuplicateStrategy = "skip"
	DuplicateUpdate DuplicateStrategy = "update"
	DuplicateCreate DuplicateStrategy = "create"

	StatusNew       EntryStatus = "new"
	StatusDuplicate EntryStatus = "duplicate"
	StatusInvalid   EntryStatus = "invalid"

	MappingNew      CategoryMapping = "new"
	MappingExisting CategoryMapping = "existing"
)

type Link struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	URL         string                `json:"url"`
	IconSource  navigation.IconSource `json:"iconSource"`
	IconValue   string                `json:"iconValue"`
	Problem     string                `json:"problem,omitempty"`
}

type Category struct {
	Name       string                `json:"name"`
	IconName   string                `json:"iconName"`
	Visibility navigation.Visibility `json:"visibility"`
	Links      []Link                `json:"links"`
}

type Batch struct {
	Format     Format     `json:"format"`
	Categories []Category `json:"categories"`
}

type PreviewLink struct {
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Status      EntryStatus `json:"status"`
	Problem     string      `json:"problem,omitempty"`
	DuplicateOf string      `json:"duplicateOf,omitempty"`
}

type PreviewCategory struct {
	Name       string          `json:"name"`
	Mapping    CategoryMapping `json:"mapping"`
	ExistingID string          `json:"existingId,omitempty"`
	Links      []PreviewLink   `json:"links"`
}

type PreviewSummary struct {
	New                int `json:"new"`
	Duplicates         int `json:"duplicates"`
	Invalid            int `json:"invalid"`
	NewCategories      int `json:"newCategories"`
	ExistingCategories int `json:"existingCategories"`
}

type Preview struct {
	Format     Format                `json:"format"`
	Visibility navigation.Visibility `json:"visibility"`
	Summary    PreviewSummary        `json:"summary"`
	Categories []PreviewCategory     `json:"categories"`
}

type ImportOptions struct {
	Visibility navigation.Visibility
	Duplicates DuplicateStrategy
}

type CommitResult struct {
	CreatedCategories int     `json:"createdCategories"`
	CreatedLinks      int     `json:"createdLinks"`
	UpdatedLinks      int     `json:"updatedLinks"`
	SkippedLinks      int     `json:"skippedLinks"`
	InvalidLinks      int     `json:"invalidLinks"`
	ImportedLinkIDs   []int64 `json:"-"`
}

type Repository interface {
	Navigation(ctx context.Context, includePrivate bool) ([]navigation.Group, error)
	ImportBookmarks(ctx context.Context, batch Batch, options ImportOptions) (CommitResult, error)
}
