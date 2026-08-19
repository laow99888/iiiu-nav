package server

import (
	"errors"
	"fmt"
	"net/http"

	"iiiu-nav/internal/backup"
)

type backupHandler struct {
	service *backup.Service
}

func registerBackupRoutes(mux *http.ServeMux, authenticator Authenticator, service *backup.Service) {
	handler := &backupHandler{service: service}
	mux.Handle("GET /api/backups", RequireAdmin(authenticator, http.HandlerFunc(handler.list)))
	mux.Handle("POST /api/backups", RequireAdmin(authenticator, http.HandlerFunc(handler.create)))
	mux.Handle("GET /api/backups/{name}", RequireAdmin(authenticator, http.HandlerFunc(handler.download)))
	mux.Handle("DELETE /api/backups/{name}", RequireAdmin(authenticator, http.HandlerFunc(handler.delete)))
}

func (handler *backupHandler) list(writer http.ResponseWriter, _ *http.Request) {
	backups, err := handler.service.List()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "backup_list_failed")
		return
	}
	writeJSON(writer, http.StatusOK, map[string][]backup.Info{"backups": backups})
}

func (handler *backupHandler) create(writer http.ResponseWriter, request *http.Request) {
	created, err := handler.service.Create(request.Context())
	if errors.Is(err, backup.ErrInsufficientSpace) {
		writeError(writer, http.StatusInsufficientStorage, "backup_space_insufficient")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "backup_create_failed")
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (handler *backupHandler) download(writer http.ResponseWriter, request *http.Request) {
	file, info, err := handler.service.Open(request.PathValue("name"))
	if errors.Is(err, backup.ErrBackupNotFound) {
		writeError(writer, http.StatusNotFound, "backup_not_found")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "backup_download_failed")
		return
	}
	defer file.Close()
	writer.Header().Set("Content-Type", "application/zip")
	writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.Name()))
	http.ServeContent(writer, request, info.Name(), info.ModTime(), file)
}

func (handler *backupHandler) delete(writer http.ResponseWriter, request *http.Request) {
	err := handler.service.Delete(request.PathValue("name"))
	if errors.Is(err, backup.ErrBackupNotFound) {
		writeError(writer, http.StatusNotFound, "backup_not_found")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "backup_delete_failed")
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
