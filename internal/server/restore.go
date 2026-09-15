package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/restore"
)

const restoreRequestOverhead = 1 << 20

type PasswordVerifier interface {
	VerifyPassword(ctx context.Context, password string) error
}

type restoreHandler struct {
	verifier PasswordVerifier
	service  *restore.Service
	gate     *maintenanceGate
}

func registerRestoreRoutes(mux *http.ServeMux, authenticator Authenticator, verifier PasswordVerifier, service *restore.Service, gate *maintenanceGate) {
	handler := &restoreHandler{verifier: verifier, service: service, gate: gate}
	mux.Handle("POST /api/restore", RequireAdmin(authenticator, http.HandlerFunc(handler.restore)))
}

func (handler *restoreHandler) restore(writer http.ResponseWriter, request *http.Request) {
	clearResponseDeadline(writer)
	request.Body = http.MaxBytesReader(writer, request.Body, restore.MaxCompressedBytes+restoreRequestOverhead)
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(writer, http.StatusRequestEntityTooLarge, "restore_archive_too_large")
		} else {
			writeError(writer, http.StatusBadRequest, "restore_request_invalid")
		}
		return
	}
	defer request.MultipartForm.RemoveAll()
	if request.FormValue("confirmation") != "RESTORE" {
		writeError(writer, http.StatusUnprocessableEntity, "restore_confirmation_required")
		return
	}
	if err := handler.verifier.VerifyPassword(request.Context(), request.FormValue("password")); errors.Is(err, auth.ErrInvalidCredentials) {
		writeError(writer, http.StatusUnauthorized, "restore_password_invalid")
		return
	} else if err != nil {
		writeError(writer, http.StatusInternalServerError, "restore_verification_failed")
		return
	}
	file, header, err := request.FormFile("backup")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "restore_archive_required")
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > restore.MaxCompressedBytes {
		writeError(writer, http.StatusRequestEntityTooLarge, "restore_archive_too_large")
		return
	}
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		writeError(writer, http.StatusUnprocessableEntity, "restore_archive_invalid")
		return
	}
	handler.gate.Lock()
	defer handler.gate.Unlock()
	// A client disconnect must not cancel the swap: after the active database
	// is closed, a canceled context would leave the store permanently closed.
	result, err := handler.service.Restore(context.WithoutCancel(request.Context()), file, header.Size)
	switch {
	case errors.Is(err, restore.ErrArchiveTooLarge):
		writeError(writer, http.StatusRequestEntityTooLarge, "restore_archive_too_large")
	case errors.Is(err, restore.ErrArchiveIncompatible):
		writeError(writer, http.StatusUnprocessableEntity, "restore_archive_incompatible")
	case errors.Is(err, restore.ErrArchiveInvalid):
		writeError(writer, http.StatusUnprocessableEntity, "restore_archive_invalid")
	case errors.Is(err, restore.ErrInsufficientSpace):
		writeError(writer, http.StatusInsufficientStorage, "restore_space_insufficient")
	case err != nil:
		writeError(writer, http.StatusInternalServerError, "restore_failed")
	default:
		clearSessionCookie(writer, request)
		writeJSON(writer, http.StatusOK, result)
	}
}
