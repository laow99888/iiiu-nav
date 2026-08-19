package appdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultRoot = "data"

type Layout struct {
	Root        string
	Database    string
	Uploads     string
	Logos       string
	Backgrounds string
	Site        string
	Backups     string
}

func Prepare(root string) (Layout, error) {
	if root == "" {
		return Layout{}, errors.New("data root is required")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Layout{}, fmt.Errorf("resolve data root: %w", err)
	}

	layout := Layout{
		Root:        absoluteRoot,
		Database:    filepath.Join(absoluteRoot, "nav.db"),
		Uploads:     filepath.Join(absoluteRoot, "uploads"),
		Logos:       filepath.Join(absoluteRoot, "uploads", "logos"),
		Backgrounds: filepath.Join(absoluteRoot, "uploads", "backgrounds"),
		Site:        filepath.Join(absoluteRoot, "uploads", "site"),
		Backups:     filepath.Join(absoluteRoot, "backups"),
	}

	for _, directory := range []string{
		layout.Root,
		layout.Uploads,
		layout.Logos,
		layout.Backgrounds,
		layout.Site,
		layout.Backups,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return Layout{}, fmt.Errorf("create data directory %q: %w", directory, err)
		}
	}

	return layout, nil
}
