package backup

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var backupNamePattern = regexp.MustCompile(`^iiiu-nav-backup-\d{8}-\d{6}-[a-f0-9]{8}\.zip$`)

type Service struct {
	config Config
	now    func() time.Time
}

func New(config Config) (*Service, error) {
	if config.Root == "" || config.DatabasePath == "" || config.UploadsPath == "" || config.Database == nil {
		return nil, errors.New("backup paths and database are required")
	}
	if config.AvailableSpace == nil {
		config.AvailableSpace = availableDiskSpace
	}
	return &Service{config: config, now: time.Now}, nil
}

func (service *Service) Create(ctx context.Context) (Info, error) {
	required, err := service.requiredSpace()
	if err != nil {
		return Info{}, err
	}
	available, err := service.config.AvailableSpace(service.config.Root)
	if err != nil {
		return Info{}, fmt.Errorf("read backup free space: %w", err)
	}
	if available < required {
		return Info{}, ErrInsufficientSpace
	}
	working, err := os.MkdirTemp(service.config.Root, ".backup-creating-")
	if err != nil {
		return Info{}, fmt.Errorf("create backup workspace: %w", err)
	}
	defer os.RemoveAll(working)

	createdAt := service.now().UTC()
	snapshotPath := filepath.Join(working, "nav.db")
	schemaVersion, err := service.config.Database.BackupSnapshot(ctx, snapshotPath)
	if err != nil {
		return Info{}, fmt.Errorf("create database snapshot: %w", err)
	}
	archivePath := filepath.Join(working, "backup.zip")
	if err := service.writeArchive(ctx, archivePath, snapshotPath, schemaVersion, createdAt); err != nil {
		return Info{}, err
	}
	name, err := backupName(createdAt)
	if err != nil {
		return Info{}, fmt.Errorf("create backup name: %w", err)
	}
	finalPath := filepath.Join(service.config.Root, name)
	if err := os.Rename(archivePath, finalPath); err != nil {
		return Info{}, fmt.Errorf("publish backup: %w", err)
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		return Info{}, fmt.Errorf("read created backup: %w", err)
	}
	return Info{Name: name, Size: info.Size(), CreatedAt: createdAt}, nil
}

func (service *Service) List() ([]Info, error) {
	entries, err := os.ReadDir(service.config.Root)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	result := make([]Info, 0)
	for _, entry := range entries {
		if entry.IsDir() || !backupNamePattern.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		result = append(result, Info{Name: entry.Name(), Size: info.Size(), CreatedAt: info.ModTime().UTC()})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name > result[right].Name })
	return result, nil
}

func (service *Service) Open(name string) (*os.File, os.FileInfo, error) {
	path, err := service.path(name)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, ErrBackupNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("open backup: %w", err)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, nil, ErrBackupNotFound
	}
	return file, info, nil
}

func (service *Service) Delete(name string) error {
	path, err := service.path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
		return ErrBackupNotFound
	} else if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	return nil
}

func (service *Service) path(name string) (string, error) {
	if !backupNamePattern.MatchString(name) || filepath.Base(name) != name {
		return "", ErrBackupNotFound
	}
	return filepath.Join(service.config.Root, name), nil
}

func (service *Service) requiredSpace() (uint64, error) {
	database, err := os.Stat(service.config.DatabasePath)
	if err != nil {
		return 0, fmt.Errorf("read database size: %w", err)
	}
	var uploads int64
	err = filepath.WalkDir(service.config.UploadsPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		uploads += info.Size()
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("measure uploads: %w", err)
	}
	databaseFootprint := database.Size()
	for _, suffix := range []string{"-wal", "-shm"} {
		if companion, err := os.Stat(service.config.DatabasePath + suffix); err == nil {
			databaseFootprint += companion.Size()
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("read database companion size: %w", err)
		}
	}
	required := databaseFootprint*2 + uploads + (16 << 20)
	return uint64(max(required, 0)), nil
}

func backupName(createdAt time.Time) (string, error) {
	random := make([]byte, 4)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("iiiu-nav-backup-%s-%s.zip", createdAt.Format("20060102-150405"), hex.EncodeToString(random)), nil
}

func (service *Service) writeArchive(ctx context.Context, destination, snapshot string, schemaVersion int, createdAt time.Time) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create backup archive: %w", err)
	}
	archive := zip.NewWriter(file)
	manifest := Manifest{Format: ManifestFormat, Version: ManifestVersion, ApplicationVersion: service.config.ApplicationVersion, SchemaVersion: schemaVersion, CreatedAt: createdAt, Files: make([]FileIntegrity, 0)}
	writeErr := addArchiveFile(ctx, archive, snapshot, "nav.db", &manifest)
	if writeErr == nil {
		writeErr = addUploads(ctx, archive, service.config.UploadsPath, &manifest)
	}
	if writeErr == nil {
		writeErr = addManifest(archive, manifest)
	}
	closeArchiveErr := archive.Close()
	closeFileErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeArchiveErr != nil {
		return fmt.Errorf("finish backup archive: %w", closeArchiveErr)
	}
	if closeFileErr != nil {
		return fmt.Errorf("close backup archive: %w", closeFileErr)
	}
	return nil
}

func addUploads(ctx context.Context, archive *zip.Writer, root string, manifest *Manifest) error {
	if _, err := archive.CreateHeader(&zip.FileHeader{Name: "uploads/", Method: zip.Store}); err != nil {
		return fmt.Errorf("add uploads directory: %w", err)
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := "uploads/" + filepath.ToSlash(relative)
		if entry.IsDir() {
			_, err := archive.CreateHeader(&zip.FileHeader{Name: strings.TrimSuffix(name, "/") + "/", Method: zip.Store})
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return addArchiveFile(ctx, archive, path, name, manifest)
	})
}

func addArchiveFile(ctx context.Context, archive *zip.Writer, source, name string, manifest *Manifest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open backup input %s: %w", name, err)
	}
	defer input.Close()
	entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
	if err != nil {
		return fmt.Errorf("create backup entry %s: %w", name, err)
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(entry, hash), input)
	if err != nil {
		return fmt.Errorf("write backup entry %s: %w", name, err)
	}
	manifest.Files = append(manifest.Files, FileIntegrity{Path: name, Size: written, SHA256: hex.EncodeToString(hash.Sum(nil))})
	return nil
}

func addManifest(archive *zip.Writer, manifest Manifest) error {
	entry, err := archive.CreateHeader(&zip.FileHeader{Name: "manifest.json", Method: zip.Deflate})
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(entry)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
}
