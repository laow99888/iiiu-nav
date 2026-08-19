package backup

import (
	"context"
	"errors"
	"time"
)

const (
	ManifestFormat  = "iiiu-nav-backup"
	ManifestVersion = 1
)

var (
	ErrInsufficientSpace = errors.New("insufficient free space for backup")
	ErrBackupNotFound    = errors.New("backup not found")
)

type Database interface {
	BackupSnapshot(ctx context.Context, destination string) (schemaVersion int, err error)
}

type FileIntegrity struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Format             string          `json:"format"`
	Version            int             `json:"version"`
	ApplicationVersion string          `json:"applicationVersion"`
	SchemaVersion      int             `json:"schemaVersion"`
	CreatedAt          time.Time       `json:"createdAt"`
	Files              []FileIntegrity `json:"files"`
}

type Info struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

type Config struct {
	Root               string
	DatabasePath       string
	UploadsPath        string
	ApplicationVersion string
	Database           Database
	AvailableSpace     func(string) (uint64, error)
}
