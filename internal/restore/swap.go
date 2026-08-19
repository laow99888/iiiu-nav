package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (service *Service) swap(ctx context.Context, stagedDatabase, stagedUploads string) error {
	rollback, err := os.MkdirTemp(service.config.DataRoot, ".restore-rollback-")
	if err != nil {
		return fmt.Errorf("create restore rollback workspace: %w", err)
	}
	defer os.RemoveAll(rollback)
	if err := service.config.Database.Close(); err != nil {
		return fmt.Errorf("close active database for restore: %w", err)
	}
	rollbackErr := func(cause error) error {
		if err := service.rollbackSwap(ctx, rollback); err != nil {
			return fmt.Errorf("restore failed (%v) and rollback failed: %w", cause, err)
		}
		return cause
	}
	if err := service.moveIfExists(service.config.DatabasePath, filepath.Join(rollback, "nav.db")); err != nil {
		return rollbackErr(fmt.Errorf("stage active database: %w", err))
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := service.moveIfExists(service.config.DatabasePath+suffix, filepath.Join(rollback, "nav.db"+suffix)); err != nil {
			return rollbackErr(fmt.Errorf("stage active database companion: %w", err))
		}
	}
	if err := service.moveIfExists(service.config.UploadsPath, filepath.Join(rollback, "uploads")); err != nil {
		return rollbackErr(fmt.Errorf("stage active uploads: %w", err))
	}
	if err := service.config.Rename(stagedDatabase, service.config.DatabasePath); err != nil {
		return rollbackErr(fmt.Errorf("activate restored database: %w", err))
	}
	if err := service.config.Rename(stagedUploads, service.config.UploadsPath); err != nil {
		return rollbackErr(fmt.Errorf("activate restored uploads: %w", err))
	}
	if err := service.config.Database.Reopen(ctx); err != nil {
		return rollbackErr(fmt.Errorf("reopen restored database: %w", err))
	}
	return nil
}

func (service *Service) rollbackSwap(ctx context.Context, rollback string) error {
	failed := filepath.Join(rollback, "failed")
	if err := os.MkdirAll(failed, 0o700); err != nil {
		return err
	}
	if pathExists(filepath.Join(rollback, "nav.db")) {
		_ = service.moveIfExists(service.config.DatabasePath, filepath.Join(failed, "nav.db"))
		for _, suffix := range []string{"-wal", "-shm"} {
			_ = service.moveIfExists(service.config.DatabasePath+suffix, filepath.Join(failed, "nav.db"+suffix))
		}
		if err := service.moveIfExists(filepath.Join(rollback, "nav.db"), service.config.DatabasePath); err != nil {
			return err
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			if err := service.moveIfExists(filepath.Join(rollback, "nav.db"+suffix), service.config.DatabasePath+suffix); err != nil {
				return err
			}
		}
	}
	if pathExists(filepath.Join(rollback, "uploads")) {
		_ = service.moveIfExists(service.config.UploadsPath, filepath.Join(failed, "uploads"))
		if err := service.moveIfExists(filepath.Join(rollback, "uploads"), service.config.UploadsPath); err != nil {
			return err
		}
	}
	return service.config.Database.Reopen(ctx)
}

func pathExists(value string) bool {
	_, err := os.Stat(value)
	return err == nil
}

func (service *Service) moveIfExists(source, destination string) error {
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return service.config.Rename(source, destination)
}
