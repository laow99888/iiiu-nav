package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"iiiu-nav/internal/auth"
	"iiiu-nav/internal/navigation"
	"iiiu-nav/internal/siteconfig"
)

type NavigationReader interface {
	Navigation(ctx context.Context, includePrivate bool) ([]navigation.Group, error)
}

type navigationHandler struct {
	authenticator Authenticator
	reader        NavigationReader
	settings      SettingsStore
}

type navigationResponse struct {
	Administrator bool                         `json:"administrator"`
	Categories    []navigationCategoryResponse `json:"categories"`
	SearchEngines []searchEngineConfig         `json:"searchEngines"`
	Site          siteconfig.Settings          `json:"site"`
}

type navigationCategoryResponse struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Slug       string                   `json:"slug"`
	IconName   string                   `json:"iconName"`
	Visibility navigation.Visibility    `json:"visibility"`
	Links      []navigationLinkResponse `json:"links"`
}

type navigationLinkResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	URL         string                `json:"url"`
	IconSource  navigation.IconSource `json:"iconSource"`
	IconValue   string                `json:"iconValue"`
}

func registerNavigationRoutes(mux *http.ServeMux, handler *navigationHandler) {
	mux.HandleFunc("GET /api/navigation", handler.read)
}

func (handler *navigationHandler) read(writer http.ResponseWriter, request *http.Request) {
	administrator, invalidSession, err := handler.administratorStatus(request)
	if err != nil {
		setPrivateResponseHeaders(writer)
		writeError(writer, http.StatusInternalServerError, "navigation_authentication_failed")
		return
	}
	if invalidSession {
		clearSessionCookie(writer, request)
	}
	writer.Header().Add("Vary", "Cookie")
	if administrator {
		setPrivateResponseHeaders(writer)
	} else {
		writer.Header().Set("Cache-Control", "public, max-age=60")
	}

	includePrivate := administrator && request.URL.Query().Get("scope") != "public"
	groups, err := handler.reader.Navigation(request.Context(), includePrivate)
	if err != nil {
		setPrivateResponseHeaders(writer)
		writeError(writer, http.StatusInternalServerError, "navigation_read_failed")
		return
	}
	searchEngines, err := readSearchEngineConfig(request.Context(), handler.settings)
	if err != nil {
		setPrivateResponseHeaders(writer)
		writeError(writer, http.StatusInternalServerError, "search_engines_read_failed")
		return
	}
	site, err := readSiteSettings(request.Context(), handler.settings)
	if err != nil {
		setPrivateResponseHeaders(writer)
		writeError(writer, http.StatusInternalServerError, "site_settings_read_failed")
		return
	}
	writeJSON(writer, http.StatusOK, navigationResponse{
		Administrator: administrator,
		Categories:    navigationCategories(groups),
		SearchEngines: searchEngines,
		Site:          site,
	})
}

func (handler *navigationHandler) administratorStatus(request *http.Request) (bool, bool, error) {
	if handler.authenticator == nil {
		return false, false, nil
	}
	token := sessionToken(request)
	if token == "" {
		return false, false, nil
	}
	err := handler.authenticator.Authenticate(request.Context(), token)
	if errors.Is(err, auth.ErrUnauthenticated) {
		return false, true, nil
	}
	return err == nil, false, err
}

func navigationCategories(groups []navigation.Group) []navigationCategoryResponse {
	categories := make([]navigationCategoryResponse, 0, len(groups))
	for _, group := range groups {
		links := make([]navigationLinkResponse, 0, len(group.Links))
		for _, link := range group.Links {
			links = append(links, navigationLinkResponse{
				ID:          strconv.FormatInt(link.ID, 10),
				Name:        link.Name,
				Description: link.Description,
				URL:         link.URL,
				IconSource:  link.IconSource,
				IconValue:   link.IconValue,
			})
		}
		categories = append(categories, navigationCategoryResponse{
			ID:         strconv.FormatInt(group.Category.ID, 10),
			Name:       group.Category.Name,
			Slug:       group.Category.Slug,
			IconName:   group.Category.IconName,
			Visibility: group.Category.Visibility,
			Links:      links,
		})
	}
	return categories
}
