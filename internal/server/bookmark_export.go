package server

import (
	"errors"
	"net/http"

	"iiiu-nav/internal/bookmarks"
)

func (handler *bookmarkImportHandler) export(writer http.ResponseWriter, request *http.Request) {
	scope := bookmarks.ExportScope(request.URL.Query().Get("scope"))
	if scope == "" {
		scope = bookmarks.ScopeAll
	}
	format := request.URL.Query().Get("format")
	var content []byte
	var err error
	switch format {
	case "html":
		content, err = handler.service.ExportHTML(request.Context(), scope)
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("Content-Disposition", `attachment; filename="iiiu-nav-bookmarks.html"`)
	case "json":
		content, err = handler.service.ExportJSON(request.Context(), scope)
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Content-Disposition", `attachment; filename="iiiu-nav-bookmarks.json"`)
	default:
		writeError(writer, http.StatusBadRequest, "bookmark_export_format_invalid")
		return
	}
	if errors.Is(err, bookmarks.ErrInvalidExportScope) {
		writeError(writer, http.StatusUnprocessableEntity, "bookmark_export_scope_invalid")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "bookmark_export_failed")
		return
	}
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(content)
}
