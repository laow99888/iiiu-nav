package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/navigation"
)

type LinkManager interface {
	LinksByCategory(ctx context.Context, categoryID int64) ([]navigation.Link, error)
	CreateLink(ctx context.Context, input navigation.LinkInput) (navigation.Link, error)
	UpdateLink(ctx context.Context, id int64, input navigation.LinkInput) (navigation.Link, error)
	DeleteLink(ctx context.Context, id int64) error
	ReorderLinks(ctx context.Context, categoryID int64, ids []int64) error
	Link(ctx context.Context, id int64) (navigation.Link, error)
	Links(ctx context.Context) ([]navigation.Link, error)
	LogoReferenceCount(ctx context.Context, publicPath string) (int, error)
}

type linkMutationRequest struct {
	CategoryID  string                `json:"categoryId"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	URL         string                `json:"url"`
	LogoText    string                `json:"logoText"`
	IconSource  navigation.IconSource `json:"iconSource"`
	IconValue   string                `json:"iconValue"`
}

type linkHandler struct {
	store LinkManager
	logos *imagestore.Store
}

func registerLinkRoutes(mux *http.ServeMux, authenticator Authenticator, store LinkManager, logos *imagestore.Store) {
	handler := &linkHandler{store: store, logos: logos}
	mux.Handle("POST /api/links", RequireAdmin(authenticator, http.HandlerFunc(handler.create)))
	mux.Handle("PUT /api/links/{id}", RequireAdmin(authenticator, http.HandlerFunc(handler.update)))
	mux.Handle("DELETE /api/links/{id}", RequireAdmin(authenticator, http.HandlerFunc(handler.delete)))
	mux.Handle("PUT /api/categories/{id}/links/order", RequireAdmin(authenticator, http.HandlerFunc(handler.reorder)))
}

func (handler *linkHandler) create(writer http.ResponseWriter, request *http.Request) {
	input, ok := decodeLinkInput(writer, request)
	if !ok {
		return
	}
	links, err := handler.store.LinksByCategory(request.Context(), input.CategoryID)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "link_category_invalid")
		return
	}
	input.SortOrder = (len(links) + 1) * 10
	link, err := handler.store.CreateLink(request.Context(), input)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "link_create_failed")
		return
	}
	writeJSON(writer, http.StatusCreated, linkResponseFromRecord(link))
}

func (handler *linkHandler) update(writer http.ResponseWriter, request *http.Request) {
	id, ok := positiveID(writer, request.PathValue("id"), "link_id_invalid")
	if !ok {
		return
	}
	input, ok := decodeLinkInput(writer, request)
	if !ok {
		return
	}
	previous, previousErr := handler.store.Link(request.Context(), id)
	if previousErr != nil && !errors.Is(previousErr, navigation.ErrLinkNotFound) {
		writeError(writer, http.StatusInternalServerError, "link_read_failed")
		return
	}
	link, err := handler.store.UpdateLink(request.Context(), id, input)
	if errors.Is(err, navigation.ErrLinkNotFound) {
		writeError(writer, http.StatusNotFound, "link_not_found")
	} else if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "link_update_failed")
	} else {
		handler.removeUnreferencedLogo(request.Context(), previous.IconValue, link.IconValue)
		writeJSON(writer, http.StatusOK, linkResponseFromRecord(link))
	}
}

func (handler *linkHandler) delete(writer http.ResponseWriter, request *http.Request) {
	id, ok := positiveID(writer, request.PathValue("id"), "link_id_invalid")
	if !ok {
		return
	}
	previous, previousErr := handler.store.Link(request.Context(), id)
	if previousErr != nil && !errors.Is(previousErr, navigation.ErrLinkNotFound) {
		writeError(writer, http.StatusInternalServerError, "link_read_failed")
		return
	}
	err := handler.store.DeleteLink(request.Context(), id)
	if errors.Is(err, navigation.ErrLinkNotFound) {
		writeError(writer, http.StatusNotFound, "link_not_found")
	} else if err != nil {
		writeError(writer, http.StatusInternalServerError, "link_delete_failed")
	} else {
		handler.removeUnreferencedLogo(request.Context(), previous.IconValue, "")
		writer.WriteHeader(http.StatusNoContent)
	}
}

func (handler *linkHandler) reorder(writer http.ResponseWriter, request *http.Request) {
	categoryID, ok := positiveID(writer, request.PathValue("id"), "category_id_invalid")
	if !ok {
		return
	}
	var body categoryOrderRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	ids := make([]int64, 0, len(body.IDs))
	for _, value := range body.IDs {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			writeError(writer, http.StatusBadRequest, "link_order_invalid")
			return
		}
		ids = append(ids, id)
	}
	if err := handler.store.ReorderLinks(request.Context(), categoryID, ids); errors.Is(err, navigation.ErrInvalidLinkOrder) {
		writeError(writer, http.StatusConflict, "link_order_stale")
	} else if err != nil {
		writeError(writer, http.StatusInternalServerError, "link_reorder_failed")
	} else {
		writer.WriteHeader(http.StatusNoContent)
	}
}

func decodeLinkInput(writer http.ResponseWriter, request *http.Request) (navigation.LinkInput, bool) {
	var body linkMutationRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return navigation.LinkInput{}, false
	}
	categoryID, err := strconv.ParseInt(body.CategoryID, 10, 64)
	parsedURL, urlErr := url.ParseRequestURI(strings.TrimSpace(body.URL))
	name, description, logoText := strings.TrimSpace(body.Name), strings.TrimSpace(body.Description), strings.TrimSpace(body.LogoText)
	if err != nil || categoryID <= 0 || urlErr != nil || parsedURL.Host == "" || parsedURL.User != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || name == "" || utf8.RuneCountInString(name) > 120 || utf8.RuneCountInString(description) > 300 || utf8.RuneCountInString(logoText) > 3 {
		writeError(writer, http.StatusUnprocessableEntity, "link_invalid")
		return navigation.LinkInput{}, false
	}
	iconSource, iconValue := body.IconSource, strings.TrimSpace(body.IconValue)
	if iconSource == "" {
		iconSource, iconValue = navigation.IconSourceGenerated, logoText
	}
	if iconSource == navigation.IconSourceGenerated {
		if iconValue == "" {
			iconValue = firstRune(name)
		}
		if utf8.RuneCountInString(iconValue) > 3 {
			writeError(writer, http.StatusUnprocessableEntity, "link_invalid")
			return navigation.LinkInput{}, false
		}
	} else if (iconSource != navigation.IconSourceAuto && iconSource != navigation.IconSourceUpload && iconSource != navigation.IconSourceURL) || !validLogoPath(iconValue) {
		writeError(writer, http.StatusUnprocessableEntity, "link_invalid")
		return navigation.LinkInput{}, false
	}
	return navigation.LinkInput{CategoryID: categoryID, Name: name, Description: description, URL: parsedURL.String(), IconSource: iconSource, IconValue: iconValue}, true
}

func positiveID(writer http.ResponseWriter, value, code string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		writeError(writer, http.StatusBadRequest, code)
		return 0, false
	}
	return id, true
}
func firstRune(value string) string {
	runeValue, _ := utf8.DecodeRuneInString(value)
	return string(runeValue)
}
func linkResponseFromRecord(link navigation.Link) navigationLinkResponse {
	return navigationLinkResponse{ID: strconv.FormatInt(link.ID, 10), Name: link.Name, Description: link.Description, URL: link.URL, IconSource: link.IconSource, IconValue: link.IconValue}
}

func validLogoPath(value string) bool {
	_, valid := imagestore.Filename(value, imagestore.LogoPrefix)
	return valid
}

func (handler *linkHandler) removeUnreferencedLogo(ctx context.Context, previous, current string) {
	if handler.logos == nil || previous == "" || previous == current || !validLogoPath(previous) {
		return
	}
	count, err := handler.store.LogoReferenceCount(ctx, previous)
	if err == nil && count == 0 {
		_ = handler.logos.Remove(previous)
	}
}
