package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"iiiu-nav/internal/auth"
)

func resetAdminPassword(logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	password, err := auth.ReadPasswordFile(os.Getenv("ADMIN_PASSWORD_FILE"))
	if err != nil {
		return commandError("read reset password", err)
	}
	authenticator, err := auth.New(auth.Config{Store: data.store})
	if err != nil {
		return commandError("configure authentication", err)
	}
	if err := authenticator.ResetPassword(ctx, password); err != nil {
		return commandError("reset administrator password", err)
	}
	logger.Info("administrator password reset; all sessions invalidated")
	return nil
}
