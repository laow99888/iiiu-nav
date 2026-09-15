package restore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	rollbackPrefix   = ".restore-rollback-"
	validatingPrefix = ".restore-validating-"
	databaseFileName = "nav.db"
	uploadsDirName   = "uploads"
)

// RecoverInterrupted reconciles the data root after a process crashed during a
// restore. The swap in swap.go is a sequence of individual renames, so a crash
// can leave the active database missing while the previous state sits in a
// `.restore-rollback-*` directory; without recovery the next startup would
// silently create an empty database. Workspaces from crashes before the swap
// (`.restore-validating-*`) and empty rollback directories only hold extracted
// copies, so they are removed.
func RecoverInterrupted(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read data root for restore recovery: %w", err)
	}

	var validating []string
	var rollbacks []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		switch name := entry.Name(); {
		case strings.HasPrefix(name, validatingPrefix):
			validating = append(validating, filepath.Join(root, name))
		case strings.HasPrefix(name, rollbackPrefix):
			rollbacks = append(rollbacks, entry)
		}
	}

	var actions []string
	for _, directory := range validating {
		if err := os.RemoveAll(directory); err != nil {
			return actions, fmt.Errorf("remove interrupted restore workspace %s: %w", directory, err)
		}
		actions = append(actions, fmt.Sprintf("removed interrupted restore workspace %s", filepath.Base(directory)))
	}

	databasePath := filepath.Join(root, databaseFileName)
	_, activeDatabaseErr := os.Stat(databasePath)
	if activeDatabaseErr == nil {
		// The restored database was activated before the crash; rollback
		// copies hold previous states and stay for the operator to inspect.
		for _, entry := range rollbacks {
			actions = append(actions, fmt.Sprintf("found leftover restore rollback directory %s while the active database is intact; it was left untouched", entry.Name()))
		}
		return actions, nil
	}
	if !errors.Is(activeDatabaseErr, os.ErrNotExist) {
		return actions, fmt.Errorf("inspect active database for restore recovery: %w", activeDatabaseErr)
	}

	// Newest first: with several rollback candidates the most recent one is
	// the intended previous state.
	sort.Slice(rollbacks, func(left, right int) bool {
		leftInfo, leftErr := rollbacks[left].Info()
		rightInfo, rightErr := rollbacks[right].Info()
		if leftErr != nil || rightErr != nil {
			return rollbacks[left].Name() > rollbacks[right].Name()
		}
		return leftInfo.ModTime().After(rightInfo.ModTime())
	})
	for _, entry := range rollbacks {
		recovered, err := recoverRollbackDirectory(root, filepath.Join(root, entry.Name()))
		actions = append(actions, recovered...)
		if err != nil {
			return actions, err
		}
	}
	return actions, nil
}

func recoverRollbackDirectory(root, rollback string) ([]string, error) {
	previousDatabase := filepath.Join(rollback, databaseFileName)
	if _, err := os.Stat(previousDatabase); errors.Is(err, os.ErrNotExist) {
		if err := os.RemoveAll(rollback); err != nil {
			return nil, fmt.Errorf("remove empty restore rollback directory %s: %w", rollback, err)
		}
		return []string{fmt.Sprintf("removed empty restore rollback directory %s", filepath.Base(rollback))}, nil
	} else if err != nil {
		return nil, fmt.Errorf("inspect restore rollback database %s: %w", previousDatabase, err)
	}

	failed := filepath.Join(rollback, "failed")
	if err := os.MkdirAll(failed, 0o700); err != nil {
		return nil, fmt.Errorf("create restore recovery workspace: %w", err)
	}
	databasePath := filepath.Join(root, databaseFileName)
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := moveFileIfExists(databasePath+suffix, filepath.Join(failed, databaseFileName+suffix)); err != nil {
			return nil, fmt.Errorf("set aside stray database companion for restore recovery: %w", err)
		}
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := moveFileIfExists(filepath.Join(rollback, databaseFileName+suffix), databasePath+suffix); err != nil {
			return nil, fmt.Errorf("restore previous database from %s: %w", rollback, err)
		}
	}
	rollbackUploads := filepath.Join(rollback, uploadsDirName)
	if info, err := os.Stat(rollbackUploads); err == nil && info.IsDir() {
		if err := replaceUploadsDirectory(rollbackUploads, filepath.Join(root, uploadsDirName)); err != nil {
			return nil, err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect restore rollback uploads %s: %w", rollbackUploads, err)
	}
	if err := os.RemoveAll(rollback); err != nil {
		return nil, fmt.Errorf("remove recovered restore rollback directory %s: %w", rollback, err)
	}
	return []string{fmt.Sprintf("restored the previous database and uploads from interrupted restore rollback directory %s", filepath.Base(rollback))}, nil
}

func moveFileIfExists(source, destination string) error {
	if _, err := os.Stat(source); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.Rename(source, destination)
}

func replaceUploadsDirectory(source, destination string) error {
	info, err := os.Stat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return os.Rename(source, destination)
	}
	if err != nil {
		return fmt.Errorf("inspect active uploads for restore recovery: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("active uploads path %s is not a directory", destination)
	}
	contents, err := os.ReadDir(destination)
	if err != nil {
		return fmt.Errorf("read active uploads for restore recovery: %w", err)
	}
	if len(contents) > 0 {
		return fmt.Errorf("active uploads directory %s is not empty; moved uploads from the interrupted restore were left in place", destination)
	}
	if err := os.Remove(destination); err != nil {
		return fmt.Errorf("remove empty active uploads for restore recovery: %w", err)
	}
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("restore previous uploads from the interrupted restore: %w", err)
	}
	return nil
}
