package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"iiiu-nav/internal/analytics"
	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/bookmarks"
	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/linkmeta"
	"iiiu-nav/internal/restore"
	"iiiu-nav/internal/server"
	storage "iiiu-nav/internal/storage/sqlite"
	"iiiu-nav/internal/webui"
)

func serve(logger *slog.Logger) error {
	startupContext, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStartup()

	data, err := openApplicationData(startupContext)
	if err != nil {
		return err
	}
	defer func() {
		if err := data.Close(); err != nil {
			logger.Error("database close failed", "error", err)
		}
	}()

	authenticator, err := auth.New(auth.Config{Store: data.store})
	if err != nil {
		return commandError("configure authentication", err)
	}
	created, err := authenticator.BootstrapFromFile(startupContext, os.Getenv("ADMIN_PASSWORD_FILE"))
	if err != nil {
		return commandError("bootstrap administrator", err)
	}
	if created {
		logger.Info("administrator created")
	}
	analyticsLocation, err := time.LoadLocation(environment("IIU_NAV_TIMEZONE", "Asia/Shanghai"))
	if err != nil {
		return commandError("load analytics time zone", err)
	}
	pageViews, err := analytics.New(analytics.Config{
		Location:   analyticsLocation,
		Repository: data.store,
	})
	if err != nil {
		return commandError("configure analytics", err)
	}

	schemaVersion, err := data.store.SchemaVersion(startupContext)
	if err != nil {
		return err
	}
	logger.Info("data store ready", "root", data.layout.Root, "application_version", version, "schema_version", schemaVersion)
	if data.upgrade.Backup != nil {
		logger.Info("database migration completed", "application_version", version, "from_schema", data.upgrade.FromVersion, "to_schema", data.upgrade.ToVersion, "pre_migration_backup", data.upgrade.Backup.Name)
	}
	backupManager, err := backup.New(backup.Config{
		Root: data.layout.Backups, DatabasePath: data.layout.Database,
		UploadsPath: data.layout.Uploads, ApplicationVersion: version, Database: data.store,
	})
	if err != nil {
		return commandError("configure backups", err)
	}
	restoreManager, err := restore.New(restore.Config{
		DataRoot: data.layout.Root, DatabasePath: data.layout.Database, UploadsPath: data.layout.Uploads,
		CurrentSchema: schemaVersion, Database: data.store, Backups: backupManager,
		PrepareDatabase: storage.PrepareRestoreCandidate, IncompatibleDBError: storage.ErrIncompatibleSchema,
	})
	if err != nil {
		return commandError("configure restore", err)
	}

	address := environment("IIU_NAV_ADDR", ":8080")
	httpServer := &http.Server{
		Addr: address,
		Handler: server.New(server.Config{
			Assets:      webui.Assets(),
			Auth:        authenticator,
			Categories:  data.store,
			Links:       data.store,
			Logos:       imagestore.NewLogos(data.layout.Logos),
			SiteImages:  imagestore.NewSiteLogo(data.layout.Site),
			Favicons:    imagestore.NewFavicon(data.layout.Site),
			Backgrounds: imagestore.NewBackgrounds(data.layout.Backgrounds),
			Backups:     backupManager,
			Restores:    restoreManager,
			Reverify:    authenticator,
			Imports:     bookmarks.New(data.store),
			Metadata:    linkmeta.New(),
			Navigation:  data.store,
			Settings:    data.store,
			Analytics:   pageViews,
			Version:     version,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go shutdownServer(shutdownContext, httpServer, logger)

	logger.Info("iiiu-nav listening", "address", address, "version", version)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func shutdownServer(shutdownContext context.Context, httpServer *http.Server, logger *slog.Logger) {
	<-shutdownContext.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}
}
