package main

import (
	"context"
	"fmt"

	"iiiu-nav/internal/appdata"
	"iiiu-nav/internal/backup"
	storage "iiiu-nav/internal/storage/sqlite"
)

type applicationData struct {
	layout  appdata.Layout
	store   *storage.Store
	upgrade upgradeResult
}

type upgradeResult struct {
	FromVersion int
	ToVersion   int
	Backup      *backup.Info
}

func openApplicationData(ctx context.Context) (*applicationData, error) {
	root := environment("IIU_NAV_DATA_DIR", appdata.DefaultRoot)
	layout, err := appdata.Prepare(root)
	if err != nil {
		return nil, err
	}
	return openPreparedApplicationData(ctx, layout, version)
}

func openPreparedApplicationData(ctx context.Context, layout appdata.Layout, applicationVersion string) (*applicationData, error) {
	state, err := storage.InspectSchema(ctx, layout.Database)
	if err != nil {
		return nil, fmt.Errorf("inspect database schema: %w", err)
	}
	result := upgradeResult{FromVersion: state.Version, ToVersion: storage.LatestSchemaVersion}
	if state.Exists && state.NeedsMigration {
		store, err := storage.OpenForBackup(ctx, layout.Database)
		if err != nil {
			return nil, fmt.Errorf("open database for pre-migration backup: %w", err)
		}
		manager, configureErr := backup.New(backup.Config{
			Root: layout.Backups, DatabasePath: layout.Database, UploadsPath: layout.Uploads,
			ApplicationVersion: applicationVersion, Database: store,
		})
		if configureErr != nil {
			_ = store.Close()
			return nil, fmt.Errorf("configure pre-migration backup: %w", configureErr)
		}
		created, backupErr := manager.Create(ctx)
		closeErr := store.Close()
		if backupErr != nil {
			return nil, fmt.Errorf("create pre-migration backup: %w", backupErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close database after pre-migration backup: %w", closeErr)
		}
		result.Backup = &created
	}
	store, err := storage.Open(ctx, layout.Database)
	if err != nil {
		if result.Backup != nil {
			return nil, fmt.Errorf("migrate database from schema %d to %d (pre-migration backup %s preserved): %w", state.Version, storage.LatestSchemaVersion, result.Backup.Name, err)
		}
		return nil, fmt.Errorf("initialize database schema: %w", err)
	}
	return &applicationData{layout: layout, store: store, upgrade: result}, nil
}

func (data *applicationData) Close() error {
	if err := data.store.Close(); err != nil {
		return fmt.Errorf("close application data: %w", err)
	}
	return nil
}
