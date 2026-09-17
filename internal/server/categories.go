package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"iiiu-nav/internal/imagecleanup"
	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/navigation"
)

type CategoryManager interface {
	Categories(ctx context.Context) ([]navigation.Category, error)
	CreateCategory(ctx context.Context, input navigation.CategoryInput) (navigation.Category, error)
	UpdateCategory(ctx context.Context, id int64, input navigation.CategoryInput) (navigation.Category, error)
	DeleteCategory(ctx context.Context, id int64, mode navigation.CategoryDeleteMode, targetID int64) error
	ReorderCategories(ctx context.Context, ids []int64) error
}

var supportedCategoryIcons = map[string]struct{}{
	"": {}, "folder": {}, "house": {}, "globe": {}, "code": {},
	"database": {}, "briefcase": {}, "book": {}, "graduation": {},
	"palette": {}, "film": {}, "music": {}, "game": {}, "shopping": {},
	"health": {}, "cloud": {}, "star": {}, "tools": {},
}

type categoryMutationRequest struct {
	Name       string                `json:"name"`
	IconName   string                `json:"iconName"`
	Visibility navigation.Visibility `json:"visibility"`
}

type categoryOrderRequest struct {
	IDs []string `json:"ids"`
}

type categoryHandler struct {
	store  CategoryManager
	links  LinkManager
	logos  *imagestore.Store
	logger *slog.Logger
}

func registerCategoryRoutes(mux *http.ServeMux, authenticator Authenticator, store CategoryManager, links LinkManager, logos *imagestore.Store, logger *slog.Logger) {
	handler := &categoryHandler{store: store, links: links, logos: logos, logger: logger}
	mux.Handle("POST /api/categories", RequireAdmin(authenticator, http.HandlerFunc(handler.create)))
	mux.Handle("PUT /api/categories/{id}", RequireAdmin(authenticator, http.HandlerFunc(handler.update)))
	mux.Handle("DELETE /api/categories/{id}", RequireAdmin(authenticator, http.HandlerFunc(handler.delete)))
	mux.Handle("PUT /api/categories/order", RequireAdmin(authenticator, http.HandlerFunc(handler.reorder)))
}

func (handler *categoryHandler) create(writer http.ResponseWriter, request *http.Request) {
	input, ok := decodeCategoryInput(writer, request)
	if !ok {
		return
	}
	slug, err := newCategorySlug()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "category_slug_failed")
		return
	}
	input.Slug = slug
	category, err := handler.store.CreateCategory(request.Context(), input)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "category_create_failed")
		return
	}
	writeJSON(writer, http.StatusCreated, categoryResponseFromRecord(category))
}

func (handler *categoryHandler) update(writer http.ResponseWriter, request *http.Request) {
	id, ok := categoryID(writer, request.PathValue("id"))
	if !ok {
		return
	}
	input, ok := decodeCategoryInput(writer, request)
	if !ok {
		return
	}
	category, err := handler.store.UpdateCategory(request.Context(), id, input)
	if errors.Is(err, navigation.ErrCategoryNotFound) {
		writeError(writer, http.StatusNotFound, "category_not_found")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "category_update_failed")
		return
	}
	writeJSON(writer, http.StatusOK, categoryResponseFromRecord(category))
}

func (handler *categoryHandler) delete(writer http.ResponseWriter, request *http.Request) {
	id, ok := categoryID(writer, request.PathValue("id"))
	if !ok {
		return
	}
	mode := navigation.CategoryDeleteMode(request.URL.Query().Get("links"))
	if mode != navigation.CategoryDeleteLinks && mode != navigation.CategoryMoveLinks {
		writeError(writer, http.StatusBadRequest, "category_delete_choice_required")
		return
	}
	var targetID int64
	if mode == navigation.CategoryMoveLinks {
		var targetOK bool
		targetID, targetOK = categoryID(writer, request.URL.Query().Get("target"))
		if !targetOK {
			return
		}
	}
	var deletedLinks []navigation.Link
	if mode == navigation.CategoryDeleteLinks && handler.links != nil && handler.logos != nil {
		links, listErr := handler.links.LinksByCategory(request.Context(), id)
		if listErr != nil {
			handler.logger.Warn("category logo cleanup skipped", "category_id", id, "error", listErr)
		}
		deletedLinks = links
	}
	err := handler.store.DeleteCategory(request.Context(), id, mode, targetID)
	switch {
	case errors.Is(err, navigation.ErrCategoryNotFound):
		writeError(writer, http.StatusNotFound, "category_not_found")
	case errors.Is(err, navigation.ErrCategoryHasLinks):
		writeError(writer, http.StatusConflict, "category_has_links")
	case errors.Is(err, navigation.ErrInvalidCategoryMove):
		writeError(writer, http.StatusUnprocessableEntity, "category_move_invalid")
	case err != nil:
		writeError(writer, http.StatusInternalServerError, "category_delete_failed")
	default:
		for _, link := range deletedLinks {
			imagecleanup.Retire(request.Context(), handler.logger, handler.logos, handler.links, link.IconValue)
		}
		writer.WriteHeader(http.StatusNoContent)
	}
}

func (handler *categoryHandler) reorder(writer http.ResponseWriter, request *http.Request) {
	var body categoryOrderRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	ids := make([]int64, 0, len(body.IDs))
	for _, value := range body.IDs {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			writeError(writer, http.StatusBadRequest, "category_order_invalid")
			return
		}
		ids = append(ids, id)
	}
	if err := handler.store.ReorderCategories(request.Context(), ids); errors.Is(err, navigation.ErrInvalidCategoryOrder) {
		writeError(writer, http.StatusConflict, "category_order_stale")
	} else if err != nil {
		writeError(writer, http.StatusInternalServerError, "category_reorder_failed")
	} else {
		writer.WriteHeader(http.StatusNoContent)
	}
}

func decodeCategoryInput(writer http.ResponseWriter, request *http.Request) (navigation.CategoryInput, bool) {
	var body categoryMutationRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return navigation.CategoryInput{}, false
	}
	name := strings.TrimSpace(body.Name)
	_, iconSupported := supportedCategoryIcons[body.IconName]
	if name == "" || utf8.RuneCountInString(name) > 80 || !iconSupported ||
		(body.Visibility != navigation.VisibilityPublic && body.Visibility != navigation.VisibilityPrivate) {
		writeError(writer, http.StatusUnprocessableEntity, "category_invalid")
		return navigation.CategoryInput{}, false
	}
	return navigation.CategoryInput{Name: name, IconName: body.IconName, Visibility: body.Visibility}, true
}

func categoryID(writer http.ResponseWriter, value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		writeError(writer, http.StatusBadRequest, "category_id_invalid")
		return 0, false
	}
	return id, true
}

func newCategorySlug() (string, error) {
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "category-" + hex.EncodeToString(value), nil
}

func categoryResponseFromRecord(category navigation.Category) navigationCategoryResponse {
	return navigationCategoryResponse{
		ID: categoryIDString(category.ID), Name: category.Name, Slug: category.Slug,
		IconName: category.IconName, Visibility: category.Visibility,
		Links: make([]navigationLinkResponse, 0),
	}
}

func categoryIDString(id int64) string { return strconv.FormatInt(id, 10) }
