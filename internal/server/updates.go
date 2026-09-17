package server

import (
	"context"
	"errors"
	"net/http"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/updatecheck"
	"iiiu-nav/internal/updateexecutor"
)

type UpdateChecker interface {
	Check(context.Context, bool) updatecheck.Result
}

// UpdateExecutor is the optional host-side update executor; a nil executor
// keeps the documented manual update path.
type UpdateExecutor interface {
	Available() bool
	Start(ctx context.Context, request updateexecutor.StartRequest) error
	Status(ctx context.Context) (updateexecutor.Status, error)
}

// SchemaVersioner reports the active database schema version, recorded in the
// update request so the executor can recognize schema-aware failure recovery.
type SchemaVersioner interface {
	SchemaVersion(ctx context.Context) (int, error)
}

type updateStatusResponse struct {
	updatecheck.Result
	AutomaticUpdate bool `json:"automaticUpdate"`
}

type updateHandler struct {
	checker  UpdateChecker
	executor UpdateExecutor
	backups  *backup.Service
	schemas  SchemaVersioner
	verifier PasswordVerifier
}

func registerUpdateRoutes(mux *http.ServeMux, authenticator Authenticator, verifier PasswordVerifier, checker UpdateChecker, executor UpdateExecutor, backups *backup.Service, schemas SchemaVersioner) {
	handler := &updateHandler{checker: checker, executor: executor, backups: backups, schemas: schemas, verifier: verifier}
	mux.Handle("GET /api/updates", RequireAdmin(authenticator, http.HandlerFunc(handler.status)))
	mux.Handle("POST /api/updates/check", RequireAdmin(authenticator, http.HandlerFunc(handler.checkNow)))
	if executor != nil {
		mux.Handle("GET /api/updates/install/status", RequireAdmin(authenticator, http.HandlerFunc(handler.installStatus)))
		mux.Handle("POST /api/updates/install", RequireAdmin(authenticator, http.HandlerFunc(handler.install)))
	}
}

func (handler *updateHandler) status(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, updateStatusResponse{
		Result:          handler.checker.Check(request.Context(), false),
		AutomaticUpdate: handler.executor != nil && handler.executor.Available(),
	})
}

func (handler *updateHandler) checkNow(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, updateStatusResponse{
		Result:          handler.checker.Check(request.Context(), true),
		AutomaticUpdate: handler.executor != nil && handler.executor.Available(),
	})
}

type updateInstallRequest struct {
	Password string `json:"password"`
	Version  string `json:"version"`
}

// install performs the guarded sequence: password reverification, a forced
// re-check so the target version is exactly the official latest stable
// release, a consistent pre-update backup, and only then handing the
// request to the executor.
func (handler *updateHandler) install(writer http.ResponseWriter, request *http.Request) {
	var body updateInstallRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := handler.verifier.VerifyPassword(request.Context(), body.Password); errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(writer, http.StatusUnauthorized, "update_password_invalid")
		return
	} else if err != nil {
		writeError(writer, http.StatusInternalServerError, "update_verification_failed")
		return
	}
	result := handler.checker.Check(request.Context(), true)
	if result.State != updatecheck.StateUpdateAvailable || body.Version != result.LatestVersion {
		writeError(writer, http.StatusConflict, "update_not_available")
		return
	}
	schema, err := handler.schemas.SchemaVersion(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "update_schema_read_failed")
		return
	}
	info, err := handler.backups.Create(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "update_backup_failed")
		return
	}
	// The executor replaces the running container; this response may never
	// reach the browser, which then follows progress through /install/status
	// and tolerates the temporary outage.
	err = handler.executor.Start(request.Context(), updateexecutor.StartRequest{
		Version: body.Version, CurrentSchema: schema, BackupName: info.Name,
	})
	switch {
	case errors.Is(err, updateexecutor.ErrBusy):
		writeError(writer, http.StatusConflict, "update_already_running")
	case errors.Is(err, updateexecutor.ErrInvalidVersion):
		writeError(writer, http.StatusUnprocessableEntity, "update_version_invalid")
	case err != nil:
		writeError(writer, http.StatusBadGateway, "update_start_failed")
	default:
		writeJSON(writer, http.StatusAccepted, map[string]any{"started": true, "version": body.Version, "backup": info.Name})
	}
}

func (handler *updateHandler) installStatus(writer http.ResponseWriter, request *http.Request) {
	status, err := handler.executor.Status(request.Context())
	if err != nil {
		writeError(writer, http.StatusBadGateway, "update_status_failed")
		return
	}
	writeJSON(writer, http.StatusOK, status)
}
