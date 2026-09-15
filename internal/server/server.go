package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"iiiu-nav/internal/backup"
	"iiiu-nav/internal/bookmarks"
	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/linkmeta"
	"iiiu-nav/internal/restore"
)

type Config struct {
	Assets      fs.FS
	Auth        Authenticator
	Categories  CategoryManager
	Links       LinkManager
	Logos       *imagestore.Store
	SiteImages  *imagestore.Store
	Favicons    *imagestore.Store
	Backgrounds *imagestore.Store
	Backups     *backup.Service
	Restores    *restore.Service
	Reverify    PasswordVerifier
	Imports     *bookmarks.Service
	Metadata    *linkmeta.Recognizer
	Navigation  NavigationReader
	Settings    SettingsStore
	Analytics   PageViewAnalytics
	Updates     UpdateChecker
	Version     string
	Logger      *slog.Logger
	// Background is canceled before the data store closes on shutdown so
	// detached background work (post-import metadata refresh) stops cleanly.
	Background context.Context
}

type healthResponse struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

func New(config Config) http.Handler {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	background := config.Background
	if background == nil {
		background = context.Background()
	}
	mux := http.NewServeMux()
	maintenance := &maintenanceGate{}
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /api/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(writer).Encode(healthResponse{
			Name:    "iiiu-nav",
			Status:  "ok",
			Version: config.Version,
		})
	})
	if config.Auth != nil {
		registerAuthRoutes(mux, newAuthHandler(config.Auth))
	}
	if config.Navigation != nil {
		registerNavigationRoutes(mux, &navigationHandler{
			authenticator: config.Auth,
			reader:        config.Navigation,
			settings:      config.Settings,
		})
	}
	if config.Auth != nil && config.Settings != nil {
		registerSearchSettingsRoutes(mux, config.Auth, config.Settings)
	}
	if config.Auth != nil && config.Analytics != nil {
		registerAnalyticsRoutes(mux, config.Auth, config.Analytics)
	}
	if config.Auth != nil && config.Updates != nil {
		registerUpdateRoutes(mux, config.Auth, config.Updates)
	}
	if config.Auth != nil && config.Categories != nil {
		registerCategoryRoutes(mux, config.Auth, config.Categories, config.Links, config.Logos, logger)
	}
	if config.Auth != nil && config.Links != nil {
		registerLinkRoutes(mux, config.Auth, config.Links, config.Logos)
	}
	var metadata *metadataHandler
	if config.Auth != nil && config.Links != nil && config.Logos != nil && config.Metadata != nil {
		metadata = registerMetadataRoutes(mux, config.Auth, config.Links, config.Metadata, config.Logos, background)
	}
	if config.Auth != nil && config.Imports != nil {
		registerBookmarkRoutes(mux, config.Auth, config.Imports, metadata)
	}
	if config.Auth != nil && config.Backups != nil {
		registerBackupRoutes(mux, config.Auth, config.Backups)
	}
	if config.Auth != nil && config.Restores != nil && config.Reverify != nil {
		registerRestoreRoutes(mux, config.Auth, config.Reverify, config.Restores, maintenance)
	}
	if config.Logos != nil {
		registerImageRoutes(mux, config.Logos)
	}
	if config.Auth != nil && config.Settings != nil && config.SiteImages != nil && config.Favicons != nil && config.Backgrounds != nil {
		registerSiteSettingsRoutes(mux, config.Auth, config.Settings, config.SiteImages, config.Favicons, config.Backgrounds, logger)
		registerImageRoutes(mux, config.SiteImages)
		registerImageRoutes(mux, config.Backgrounds)
	}
	mux.HandleFunc("/api/", func(writer http.ResponseWriter, request *http.Request) {
		setPrivateResponseHeaders(writer)
		http.NotFound(writer, request)
	})

	if config.Assets == nil {
		mux.HandleFunc("/", func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "frontend is served by the Vite development server", http.StatusNotFound)
		})
	} else {
		frontend := spaHandler(config.Assets)
		if config.Analytics != nil {
			frontend = trackPublicPageViews(frontend, config.Auth, config.Analytics)
		}
		mux.Handle("/", frontend)
	}

	handler := securityHeaders(sameOriginOnly(maintenanceRequests(maintenance, indexingHeaders(config.Settings, mux))))
	return logServerErrors(logger, handler)
}

// clearResponseDeadline exempts one long-running request from the server-wide
// read and write timeouts, which would otherwise drop the connection in the
// middle of large uploads, restores, or bulk refreshes.
func clearResponseDeadline(writer http.ResponseWriter) {
	controller := http.NewResponseController(writer)
	_ = controller.SetReadDeadline(time.Time{})
	_ = controller.SetWriteDeadline(time.Time{})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (record *statusRecorder) WriteHeader(status int) {
	record.status = status
	record.ResponseWriter.WriteHeader(status)
}

// Unwrap keeps http.NewResponseController working through this wrapper so
// handlers can still manage per-request deadlines.
func (record *statusRecorder) Unwrap() http.ResponseWriter {
	return record.ResponseWriter
}

func logServerErrors(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		record := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(record, request)
		if record.status >= http.StatusInternalServerError {
			logger.Warn("request failed",
				"method", request.Method,
				"path", request.URL.Path,
				"status", record.status,
			)
		}
	})
}

func spaHandler(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			http.NotFound(writer, request)
			return
		}

		assetPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if assetPath == "" || assetPath == "." || assetPath == "index.html" {
			serveIndex(writer, request, assets)
			return
		}

		if info, err := fs.Stat(assets, assetPath); err == nil && !info.IsDir() {
			if strings.HasPrefix(assetPath, "assets/") {
				writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			serveFile(writer, request, assets, assetPath)
			return
		}

		if strings.Contains(path.Base(assetPath), ".") {
			http.NotFound(writer, request)
			return
		}

		serveIndex(writer, request, assets)
	})
}

func serveIndex(writer http.ResponseWriter, request *http.Request, assets fs.FS) {
	writer.Header().Set("Cache-Control", "no-cache")
	serveFile(writer, request, assets, "index.html")
}

func serveFile(writer http.ResponseWriter, request *http.Request, assets fs.FS, name string) {
	content, err := fs.ReadFile(assets, name)
	if err != nil {
		http.NotFound(writer, request)
		return
	}

	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		writer.Header().Set("Content-Type", contentType)
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(content)))
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodHead {
		return
	}

	_, _ = writer.Write(content)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; connect-src 'self'; font-src 'self'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data: blob:; manifest-src 'self'; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		writer.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		writer.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=(), usb=()")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}
