package server

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"iiiu-nav/internal/bookmarks"
	"iiiu-nav/internal/navigation"
)

const importRequestOverhead = 1 << 20

type bookmarkImportHandler struct {
	service *bookmarks.Service
	enrich  func([]int64)
}

func registerBookmarkRoutes(mux *http.ServeMux, authenticator Authenticator, service *bookmarks.Service, metadata *metadataHandler) {
	handler := &bookmarkImportHandler{service: service}
	if metadata != nil {
		handler.enrich = metadata.refreshImported
	}
	mux.Handle("POST /api/imports/preview", RequireAdmin(authenticator, http.HandlerFunc(handler.preview)))
	mux.Handle("POST /api/imports/commit", RequireAdmin(authenticator, http.HandlerFunc(handler.commit)))
	mux.Handle("GET /api/bookmarks/export", RequireAdmin(authenticator, http.HandlerFunc(handler.export)))
}

func (handler *bookmarkImportHandler) preview(writer http.ResponseWriter, request *http.Request) {
	file, visibility, ok := importUpload(writer, request)
	if !ok {
		return
	}
	defer file.Close()
	defer request.MultipartForm.RemoveAll()
	preview, err := handler.service.Preview(request.Context(), file, visibility)
	if err != nil {
		writeImportError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, preview)
}

func (handler *bookmarkImportHandler) commit(writer http.ResponseWriter, request *http.Request) {
	file, visibility, ok := importUpload(writer, request)
	if !ok {
		return
	}
	defer file.Close()
	defer request.MultipartForm.RemoveAll()
	strategy := bookmarks.DuplicateStrategy(strings.TrimSpace(request.FormValue("duplicates")))
	if strategy == "" {
		strategy = bookmarks.DuplicateSkip
	}
	result, err := handler.service.Commit(request.Context(), file, bookmarks.ImportOptions{Visibility: visibility, Duplicates: strategy})
	if err != nil {
		writeImportError(writer, err)
		return
	}
	if handler.enrich != nil {
		handler.enrich(result.ImportedLinkIDs)
	}
	writeJSON(writer, http.StatusOK, result)
}

func importUpload(writer http.ResponseWriter, request *http.Request) (multipart.File, navigation.Visibility, bool) {
	clearResponseDeadline(writer)
	request.Body = http.MaxBytesReader(writer, request.Body, bookmarks.MaxImportBytes+importRequestOverhead)
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeError(writer, http.StatusRequestEntityTooLarge, "bookmark_import_too_large")
		} else {
			writeError(writer, http.StatusBadRequest, "bookmark_request_invalid")
		}
		return nil, "", false
	}
	file, _, err := request.FormFile("bookmarks")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "bookmark_file_required")
		return nil, "", false
	}
	visibility := navigation.Visibility(strings.TrimSpace(request.FormValue("visibility")))
	if visibility == "" {
		visibility = navigation.VisibilityPrivate
	}
	if visibility != navigation.VisibilityPrivate && visibility != navigation.VisibilityPublic {
		file.Close()
		request.MultipartForm.RemoveAll()
		writeError(writer, http.StatusUnprocessableEntity, "bookmark_options_invalid")
		return nil, "", false
	}
	return file, visibility, true
}

func writeImportError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, bookmarks.ErrUnsupportedCharset):
		writeError(writer, http.StatusUnprocessableEntity, "bookmark_charset_unsupported")
	case errors.Is(err, bookmarks.ErrEmptyImport), errors.Is(err, bookmarks.ErrUnsupportedFormat):
		writeError(writer, http.StatusUnprocessableEntity, "bookmark_format_invalid")
	case errors.Is(err, bookmarks.ErrImportTooLarge):
		writeError(writer, http.StatusRequestEntityTooLarge, "bookmark_import_too_large")
	case errors.Is(err, bookmarks.ErrInvalidOptions):
		writeError(writer, http.StatusUnprocessableEntity, "bookmark_options_invalid")
	default:
		writeError(writer, http.StatusUnprocessableEntity, "bookmark_import_invalid")
	}
}
