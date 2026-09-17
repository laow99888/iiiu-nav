package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/restore"
	storage "iiiu-nav/internal/storage/sqlite"
)

// restoreFromBackup recovers the data root from a full backup archive. It is
// the container-side primitive the update executor uses for schema-aware
// failure recovery and a standalone recovery path for operators. The path is
// resolved inside the container, so compose-based invocations pass a /data
// path such as /data/backups/<name>.zip.
func restoreFromBackup(logger *slog.Logger, archivePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	data, err := openApplicationData(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := data.Close(); err != nil {
			logger.Error("database close failed", "error", err)
		}
	}()

	schemaVersion, err := data.store.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	backupManager, err := backup.New(backup.Config{
		Root: data.layout.Backups, DatabasePath: data.layout.Database,
		UploadsPath: data.layout.Uploads, ApplicationVersion: version, Database: data.store,
	})
	if err != nil {
		return commandError("configure backups", err)
	}
	service, err := restore.New(restore.Config{
		DataRoot: data.layout.Root, DatabasePath: data.layout.Database,
		UploadsPath: data.layout.Uploads, CurrentSchema: schemaVersion, Database: data.store,
		Backups: backupManager, PrepareDatabase: storage.PrepareRestoreCandidate,
		IncompatibleDBError: storage.ErrIncompatibleSchema,
	})
	if err != nil {
		return commandError("configure restore", err)
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return commandError("open backup archive", err)
	}
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil {
		return commandError("inspect backup archive", err)
	}
	result, err := service.Restore(ctx, archive, info.Size())
	if err != nil {
		return commandError("restore backup", err)
	}
	logger.Info("backup restored", "archive", archivePath, "pre_restore_backup", result.PreRestoreBackup.Name)
	return nil
}
