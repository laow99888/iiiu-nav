package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

var version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(os.Args[1:], logger); err != nil {
		logger.Error("iiiu-nav stopped", "error", err)
		os.Exit(1)
	}
}

func run(arguments []string, logger *slog.Logger) error {
	switch {
	case len(arguments) == 0:
		return serve(logger)
	case len(arguments) == 1 && arguments[0] == "serve":
		return serve(logger)
	case len(arguments) == 1 && arguments[0] == "healthcheck":
		return healthcheck()
	case len(arguments) == 2 && arguments[0] == "admin" && arguments[1] == "reset-password":
		return resetAdminPassword(logger)
	case len(arguments) == 2 && arguments[0] == "restore":
		return restoreFromBackup(logger, arguments[1])
	default:
		return errors.New("usage: iiiu-nav [serve | healthcheck | admin reset-password | restore <archive>]")
	}
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func commandError(action string, err error) error {
	return fmt.Errorf("%s: %w", action, err)
}
