// iiiu-nav-executor is the optional restricted update executor that runs on
// the Docker host. It serves a tiny HTTP interface on a unix socket shared
// with the application container and performs exactly one operation: moving
// the fixed iiiu-nav Compose service to a verified stable release. All
// behavior lives in internal/hostexecutor; this file is transport only.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"iiiu-nav/internal/hostexecutor"
)

const defaultSocket = "/run/iiiu-nav/executor.sock"

func main() {
	logger := log.New(os.Stdout, "iiiu-nav-executor ", log.LstdFlags|log.LUTC)

	socket := environment("EXECUTOR_SOCKET", defaultSocket)
	projectDir := environment("EXECUTOR_PROJECT_DIR", "/project")
	repository := environment("EXECUTOR_REPOSITORY", "ghcr.io/laow99888/iiiu-nav")
	projectName := environment("EXECUTOR_PROJECT_NAME", "iiiu-nav")
	overrideFile := environment("EXECUTOR_OVERRIDE_FILE", hostexecutor.OverrideName)

	composeFiles := []string{"compose.yaml", overrideFile}
	if custom := os.Getenv("EXECUTOR_COMPOSE_FILES"); custom != "" {
		composeFiles = strings.Split(custom, ",")
	}

	if err := os.MkdirAll(filepath.Dir(socket), 0o770); err != nil {
		logger.Fatalf("prepare socket directory: %v", err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		logger.Fatalf("listen on %s: %v", socket, err)
	}
	defer os.Remove(socket)
	// Only processes with explicit host privilege (the executor itself) and
	// the application container's dedicated socket mount may reach this.
	if err := os.Chmod(socket, 0o660); err != nil {
		logger.Fatalf("protect socket: %v", err)
	}

	service := hostexecutor.New(hostexecutor.Config{
		Repository:   repository,
		ProjectName:  projectName,
		ProjectDir:   projectDir,
		ComposeFiles: composeFiles,
		OverrideFile: overrideFile,
		Service:      "app",
		ReleasesURL:  "https://api.github.com/repos/laow99888/iiiu-nav/releases/tags/",
	}, &shellRunner{})

	logger.Printf("listening on %s (project %s, image %s)", socket, projectDir, repository)
	if err := http.Serve(listener, handler(service, logger)); err != nil {
		logger.Fatalf("serve: %v", err)
	}
}

func handler(service *hostexecutor.Service, logger *log.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, service.Status())
	})
	mux.HandleFunc("POST /start", func(writer http.ResponseWriter, request *http.Request) {
		var body struct {
			Version       string `json:"version"`
			CurrentSchema int    `json:"currentSchema"`
			BackupName    string `json:"backupName"`
		}
		raw, err := io.ReadAll(io.LimitReader(request.Body, 4<<10))
		if err == nil {
			err = json.Unmarshal(raw, &body)
		}
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
			return
		}
		start := hostexecutor.StartRequest{Version: body.Version, CurrentSchema: body.CurrentSchema, BackupName: body.BackupName}
		switch err := service.Start(start); {
		case errors.Is(err, hostexecutor.ErrUpdateRunning):
			writeJSON(writer, http.StatusConflict, map[string]string{"error": "update_already_running"})
		case errors.Is(err, hostexecutor.ErrInvalidVersion):
			writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"error": "update_version_invalid"})
		case err != nil:
			logger.Printf("start rejected: %v", err)
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "update_start_failed"})
		default:
			logger.Printf("update to %s accepted (backup %s)", body.Version, body.BackupName)
			writeJSON(writer, http.StatusAccepted, map[string]any{"accepted": true})
		}
	})
	return mux
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// shellRunner executes allow-listed docker commands. The command set is
// fixed inside internal/hostexecutor; this adapter adds no arguments.
type shellRunner struct{}

func (*shellRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return "", fmt.Errorf("%s: %s", err, trimmed)
		}
		return "", err
	}
	return string(output), nil
}
