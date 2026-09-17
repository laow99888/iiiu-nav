package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/siteconfig"
)

type siteSettingsHandler struct {
	store       SettingsStore
	siteImages  *imagestore.Store
	favicons    *imagestore.Store
	backgrounds *imagestore.Store
	logger      *slog.Logger
}

func registerSiteSettingsRoutes(mux *http.ServeMux, authenticator Authenticator, store SettingsStore, siteImages, favicons, backgrounds *imagestore.Store, logger *slog.Logger) {
	handler := &siteSettingsHandler{store: store, siteImages: siteImages, favicons: favicons, backgrounds: backgrounds, logger: logger}
	mux.Handle("PUT /api/settings/site", RequireAdmin(authenticator, http.HandlerFunc(handler.update)))
	mux.Handle("POST /api/settings/site/logo", RequireAdmin(authenticator, http.HandlerFunc(handler.uploadLogo)))
	mux.Handle("POST /api/settings/site/favicon", RequireAdmin(authenticator, http.HandlerFunc(handler.uploadFavicon)))
	mux.Handle("POST /api/settings/site/background", RequireAdmin(authenticator, http.HandlerFunc(handler.uploadBackground)))
	mux.HandleFunc("GET /robots.txt", handler.robots)
	mux.HandleFunc("GET /favicon.ico", handler.favicon)
}

func (handler *siteSettingsHandler) update(writer http.ResponseWriter, request *http.Request) {
	var input siteconfig.Settings
	if err := decodeJSON(writer, request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	next, err := siteconfig.Validate(input)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "site_settings_invalid")
		return
	}
	previous, err := readSiteSettings(request.Context(), handler.store)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "site_settings_read_failed")
		return
	}
	content, err := json.Marshal(next)
	if err != nil || handler.store.PutSetting(request.Context(), siteconfig.SettingKey, content) != nil {
		writeError(writer, http.StatusInternalServerError, "site_settings_update_failed")
		return
	}
	handler.cleanReplaced(previous, next)
	writeJSON(writer, http.StatusOK, next)
}

func (handler *siteSettingsHandler) uploadLogo(writer http.ResponseWriter, request *http.Request) {
	handler.upload(writer, request, handler.siteImages, "site_logo_invalid")
}

func (handler *siteSettingsHandler) uploadFavicon(writer http.ResponseWriter, request *http.Request) {
	handler.upload(writer, request, handler.favicons, "favicon_invalid")
}

func (handler *siteSettingsHandler) uploadBackground(writer http.ResponseWriter, request *http.Request) {
	handler.upload(writer, request, handler.backgrounds, "background_invalid")
}

func (handler *siteSettingsHandler) upload(writer http.ResponseWriter, request *http.Request, images *imagestore.Store, errorCode string) {
	request.Body = http.MaxBytesReader(writer, request.Body, images.MaxBytes()+(64<<10))
	file, _, err := request.FormFile("image")
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, errorCode)
		return
	}
	defer file.Close()
	publicPath, err := images.Save(file)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, errorCode)
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]string{"url": publicPath})
}

func (handler *siteSettingsHandler) cleanReplaced(previous, next siteconfig.Settings) {
	retained := map[string]struct{}{next.LogoURL: {}, next.FaviconURL: {}, next.BackgroundURL: {}}
	for path, store := range map[string]*imagestore.Store{
		previous.LogoURL: handler.siteImages, previous.FaviconURL: handler.siteImages,
		previous.BackgroundURL: handler.backgrounds,
	} {
		if path == "" {
			continue
		}
		if _, keep := retained[path]; keep {
			continue
		}
		if err := store.Remove(path); err != nil {
			handler.logger.Warn("replaced site image removal failed", "path", path, "error", err)
		}
	}
	if err := handler.siteImages.Prune(retained); err != nil {
		handler.logger.Warn("site image prune failed", "error", err)
	}
	if err := handler.backgrounds.Prune(retained); err != nil {
		handler.logger.Warn("background prune failed", "error", err)
	}
}

func (handler *siteSettingsHandler) robots(writer http.ResponseWriter, request *http.Request) {
	settings, err := readSiteSettings(request.Context(), handler.store)
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.Header().Set("Cache-Control", "public, max-age=60")
	if err == nil && settings.IndexingEnabled {
		_, _ = io.WriteString(writer, "User-agent: *\nAllow: /\n")
		return
	}
	_, _ = io.WriteString(writer, "User-agent: *\nDisallow: /\n")
}

func (handler *siteSettingsHandler) favicon(writer http.ResponseWriter, request *http.Request) {
	settings, err := readSiteSettings(request.Context(), handler.store)
	if err == nil && settings.FaviconURL != "" {
		if name, valid := handler.siteImages.Filename(settings.FaviconURL); valid {
			serveStoredImage(writer, request, handler.siteImages, name)
			return
		}
	}
	writer.Header().Set("Content-Type", "image/svg+xml")
	writer.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.WriteString(writer, defaultFavicon)
}

func readSiteSettings(ctx context.Context, store SettingsStore) (siteconfig.Settings, error) {
	defaults := siteconfig.Default()
	if store == nil {
		return defaults, nil
	}
	value, found, err := store.Setting(ctx, siteconfig.SettingKey)
	if err != nil || !found {
		return defaults, err
	}
	settings := defaults
	if err := json.Unmarshal(value, &settings); err != nil {
		return siteconfig.Settings{}, err
	}
	return siteconfig.Validate(settings)
}

func registerImageRoutes(mux *http.ServeMux, images *imagestore.Store) {
	serve := func(writer http.ResponseWriter, request *http.Request) {
		publicPath := images.Prefix() + request.PathValue("name")
		name, valid := images.Filename(publicPath)
		if !valid {
			http.NotFound(writer, request)
			return
		}
		serveStoredImage(writer, request, images, name)
	}
	mux.HandleFunc("GET "+images.Prefix()+"{name}", serve)
	mux.HandleFunc("HEAD "+images.Prefix()+"{name}", serve)
}

func serveStoredImage(writer http.ResponseWriter, request *http.Request, images *imagestore.Store, name string) {
	file, err := os.Open(filepath.Join(images.Root(), name))
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(writer, request)
		return
	}
	writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	contentType := "image/png"
	if strings.EqualFold(filepath.Ext(name), ".jpg") {
		contentType = "image/jpeg"
	}
	writer.Header().Set("Content-Type", contentType)
	http.ServeContent(writer, request, name, info.ModTime(), file)
}

func indexingHeaders(store SettingsStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		settings, err := readSiteSettings(request.Context(), store)
		if err != nil || !settings.IndexingEnabled {
			writer.Header().Set("X-Robots-Tag", "noindex, nofollow")
		}
		next.ServeHTTP(writer, request)
	})
}

const defaultFavicon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="7" fill="#305880"/><circle cx="16" cy="16" r="8" fill="none" stroke="white" stroke-width="2"/><path d="m18.5 13.5-2 5-5 2 2-5 5-2Z" fill="white"/></svg>`
