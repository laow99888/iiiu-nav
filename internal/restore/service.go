package restore

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"iiiu-nav/internal/backup"
)

type Service struct {
	config Config
}

func New(config Config) (*Service, error) {
	if config.DataRoot == "" || config.DatabasePath == "" || config.UploadsPath == "" || config.CurrentSchema <= 0 || config.Database == nil || config.Backups == nil || config.PrepareDatabase == nil {
		return nil, errors.New("restore paths, schema, database, backups, and validator are required")
	}
	if config.AvailableSpace == nil {
		config.AvailableSpace = availableDiskSpace
	}
	if config.Rename == nil {
		config.Rename = os.Rename
	}
	return &Service{config: config}, nil
}

func (service *Service) Restore(ctx context.Context, source io.ReaderAt, compressedSize int64) (Result, error) {
	if compressedSize <= 0 || compressedSize > MaxCompressedBytes {
		return Result{}, ErrArchiveTooLarge
	}
	archive, err := zip.NewReader(source, compressedSize)
	if err != nil {
		return Result{}, fmt.Errorf("%w: open ZIP: %v", ErrArchiveInvalid, err)
	}
	entries, manifest, extractedSize, err := inspectArchive(archive)
	if err != nil {
		return Result{}, err
	}
	if manifest.SchemaVersion <= 0 || manifest.SchemaVersion > service.config.CurrentSchema {
		return Result{}, ErrArchiveIncompatible
	}
	available, err := service.config.AvailableSpace(service.config.DataRoot)
	if err != nil {
		return Result{}, fmt.Errorf("read restore free space: %w", err)
	}
	if available < uint64(extractedSize)+(16<<20) {
		return Result{}, ErrInsufficientSpace
	}
	working, err := os.MkdirTemp(service.config.DataRoot, ".restore-validating-")
	if err != nil {
		return Result{}, fmt.Errorf("create restore workspace: %w", err)
	}
	defer os.RemoveAll(working)
	if err := extractArchive(ctx, entries, manifest, working); err != nil {
		return Result{}, err
	}
	stagedDatabase := filepath.Join(working, "nav.db")
	schemaVersion, err := service.config.PrepareDatabase(ctx, stagedDatabase, service.config.CurrentSchema)
	if service.config.IncompatibleDBError != nil && errors.Is(err, service.config.IncompatibleDBError) {
		return Result{}, ErrArchiveIncompatible
	}
	if err != nil {
		return Result{}, fmt.Errorf("%w: validate database: %v", ErrArchiveInvalid, err)
	}
	if schemaVersion != manifest.SchemaVersion {
		return Result{}, fmt.Errorf("%w: manifest schema does not match database", ErrArchiveInvalid)
	}
	preRestore, err := service.config.Backups.Create(ctx)
	if errors.Is(err, backup.ErrInsufficientSpace) {
		return Result{}, ErrInsufficientSpace
	}
	if err != nil {
		return Result{}, fmt.Errorf("create pre-restore backup: %w", err)
	}
	if err := service.swap(ctx, stagedDatabase, filepath.Join(working, "uploads")); err != nil {
		return Result{}, err
	}
	return Result{PreRestoreBackup: preRestore}, nil
}

type inspectedEntry struct {
	file *zip.File
	name string
}

func inspectArchive(archive *zip.Reader) ([]inspectedEntry, backup.Manifest, int64, error) {
	entries := make([]inspectedEntry, 0, len(archive.File))
	seen := make(map[string]struct{}, len(archive.File))
	seenFolded := make(map[string]struct{}, len(archive.File))
	var manifestFile *zip.File
	var extracted int64
	for _, file := range archive.File {
		name := file.Name
		clean := path.Clean(name)
		if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || (name != clean && name != clean+"/") {
			return nil, backup.Manifest{}, 0, fmt.Errorf("%w: unsafe archive path", ErrArchiveInvalid)
		}
		if clean != "nav.db" && clean != "manifest.json" && clean != "uploads" && !strings.HasPrefix(clean, "uploads/") {
			return nil, backup.Manifest{}, 0, fmt.Errorf("%w: unexpected archive entry", ErrArchiveInvalid)
		}
		if !allowedArchivePath(clean, file.FileInfo().IsDir()) {
			return nil, backup.Manifest{}, 0, fmt.Errorf("%w: unsupported archive path", ErrArchiveInvalid)
		}
		if _, duplicate := seen[clean]; duplicate {
			return nil, backup.Manifest{}, 0, fmt.Errorf("%w: duplicate archive entry", ErrArchiveInvalid)
		}
		seen[clean] = struct{}{}
		folded := strings.ToLower(clean)
		if _, duplicate := seenFolded[folded]; duplicate {
			return nil, backup.Manifest{}, 0, fmt.Errorf("%w: case-colliding archive entry", ErrArchiveInvalid)
		}
		seenFolded[folded] = struct{}{}
		if !file.FileInfo().IsDir() {
			if file.UncompressedSize64 > MaxExtractedBytes || extracted > MaxExtractedBytes-int64(file.UncompressedSize64) {
				return nil, backup.Manifest{}, 0, ErrArchiveTooLarge
			}
			extracted += int64(file.UncompressedSize64)
		}
		if clean == "manifest.json" {
			manifestFile = file
		}
		entries = append(entries, inspectedEntry{file: file, name: clean})
	}
	_, hasDatabase := seen["nav.db"]
	_, hasUploads := seen["uploads"]
	if !hasDatabase || !hasUploads || manifestFile == nil {
		return nil, backup.Manifest{}, 0, fmt.Errorf("%w: required entries are missing", ErrArchiveInvalid)
	}
	manifest, err := readManifest(manifestFile)
	if err != nil {
		return nil, backup.Manifest{}, 0, err
	}
	return entries, manifest, extracted, nil
}

func allowedArchivePath(name string, directory bool) bool {
	if name == "nav.db" || name == "manifest.json" {
		return !directory
	}
	if name == "uploads" || name == "uploads/logos" || name == "uploads/backgrounds" || name == "uploads/site" {
		return directory
	}
	parts := strings.Split(name, "/")
	if directory || len(parts) != 3 || parts[0] != "uploads" {
		return false
	}
	filename := parts[2]
	if filename == "" || strings.HasPrefix(filename, ".") {
		return false
	}
	extension := strings.ToLower(filepath.Ext(filename))
	// Backgrounds may store opaque JPEG encodes; every other asset is PNG.
	switch parts[1] {
	case "backgrounds":
		if extension != ".png" && extension != ".jpg" {
			return false
		}
	case "logos", "site":
		if extension != ".png" {
			return false
		}
	default:
		return false
	}
	for _, character := range filename {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' && character != '_' && character != '.' {
			return false
		}
	}
	return true
}

func readManifest(file *zip.File) (backup.Manifest, error) {
	if file.UncompressedSize64 > 1<<20 {
		return backup.Manifest{}, fmt.Errorf("%w: manifest is too large", ErrArchiveInvalid)
	}
	reader, err := file.Open()
	if err != nil {
		return backup.Manifest{}, fmt.Errorf("%w: open manifest", ErrArchiveInvalid)
	}
	defer reader.Close()
	decoder := json.NewDecoder(io.LimitReader(reader, (1<<20)+1))
	decoder.DisallowUnknownFields()
	var manifest backup.Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return backup.Manifest{}, fmt.Errorf("%w: parse manifest: %v", ErrArchiveInvalid, err)
	}
	if manifest.Format != backup.ManifestFormat || manifest.Version != backup.ManifestVersion || manifest.ApplicationVersion == "" || manifest.CreatedAt.IsZero() {
		return backup.Manifest{}, ErrArchiveIncompatible
	}
	return manifest, nil
}

func extractArchive(ctx context.Context, entries []inspectedEntry, manifest backup.Manifest, destination string) error {
	expected := make(map[string]backup.FileIntegrity, len(manifest.Files))
	for _, file := range manifest.Files {
		if _, duplicate := expected[file.Path]; duplicate || file.Path == "manifest.json" || file.Size < 0 || len(file.SHA256) != sha256.Size*2 {
			return fmt.Errorf("%w: invalid integrity manifest", ErrArchiveInvalid)
		}
		expected[file.Path] = file
	}
	verified := make(map[string]struct{}, len(expected))
	for _, entry := range entries {
		if entry.name == "manifest.json" {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(entry.name))
		if entry.file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return fmt.Errorf("extract restore directory: %w", err)
			}
			continue
		}
		integrity, found := expected[entry.name]
		if !found {
			return fmt.Errorf("%w: file missing from integrity manifest", ErrArchiveInvalid)
		}
		if err := extractFile(ctx, entry.file, target, integrity); err != nil {
			return err
		}
		verified[entry.name] = struct{}{}
	}
	if len(verified) != len(expected) {
		return fmt.Errorf("%w: manifest references missing files", ErrArchiveInvalid)
	}
	if err := os.MkdirAll(filepath.Join(destination, "uploads"), 0o700); err != nil {
		return err
	}
	return nil
}

func extractFile(ctx context.Context, source *zip.File, destination string, integrity backup.FileIntegrity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	input, err := source.Open()
	if err != nil {
		return fmt.Errorf("%w: open archive file", ErrArchiveInvalid)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("extract restore file: %w", err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hash), io.LimitReader(input, MaxExtractedBytes+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		return fmt.Errorf("extract restore file: %v %v", copyErr, closeErr)
	}
	if written != integrity.Size || written != int64(source.UncompressedSize64) || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), integrity.SHA256) {
		return fmt.Errorf("%w: file checksum mismatch", ErrArchiveInvalid)
	}
	return nil
}
