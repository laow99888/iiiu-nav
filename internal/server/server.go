package server

import (
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

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
	Version     string
}

type healthResponse struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

func New(config Config) http.Handler {
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
	if config.Auth != nil && config.Categories != nil {
		registerCategoryRoutes(mux, config.Auth, config.Categories, config.Links, config.Logos)
	}
	if config.Auth != nil && config.Links != nil {
		registerLinkRoutes(mux, config.Auth, config.Links, config.Logos)
	}
	var metadata *metadataHandler
	if config.Auth != nil && config.Links != nil && config.Logos != nil && config.Metadata != nil {
		metadata = registerMetadataRoutes(mux, config.Auth, config.Links, config.Metadata, config.Logos)
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
		registerSiteSettingsRoutes(mux, config.Auth, config.Settings, config.SiteImages, config.Favicons, config.Backgrounds)
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
		mux.Handle("/", spaHandler(config.Assets))
	}

	return securityHeaders(sameOriginOnly(maintenanceRequests(maintenance, indexingHeaders(config.Settings, mux))))
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
