package restore

import (
	"context"
	"errors"
	"io"

	"iiiu-nav/internal/backup"
)

const (
	MaxCompressedBytes = 256 << 20
	MaxExtractedBytes  = 512 << 20
)

var (
	ErrArchiveInvalid      = errors.New("restore archive is invalid")
	ErrArchiveTooLarge     = errors.New("restore archive exceeds size limit")
	ErrArchiveIncompatible = errors.New("restore archive is incompatible")
	ErrInsufficientSpace   = errors.New("insufficient free space for restore")
)

type Database interface {
	Close() error
	Reopen(ctx context.Context) error
}

type BackupCreator interface {
	Create(ctx context.Context) (backup.Info, error)
}

type Config struct {
	DataRoot            string
	DatabasePath        string
	UploadsPath         string
	CurrentSchema       int
	Database            Database
	Backups             BackupCreator
	PrepareDatabase     func(context.Context, string, int) (int, error)
	AvailableSpace      func(string) (uint64, error)
	IncompatibleDBError error
	Rename              func(string, string) error
}

type Result struct {
	PreRestoreBackup backup.Info `json:"preRestoreBackup"`
}

type ReaderAt interface {
	io.ReaderAt
}
