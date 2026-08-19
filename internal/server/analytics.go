package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"iiiu-nav/internal/analytics"
	"iiiu-nav/internal/auth"
)

type PageViewAnalytics interface {
	RecordPageView(ctx context.Context) error
	PageViewSeries(ctx context.Context, days int) ([]analytics.DailyPageViews, error)
}

type pageViewResponse struct {
	Series []dailyPageViewResponse `json:"series"`
}

type dailyPageViewResponse struct {
	Date  string `json:"date"`
	Views int    `json:"views"`
}

func registerAnalyticsRoutes(mux *http.ServeMux, authenticator Authenticator, service PageViewAnalytics) {
	mux.Handle("GET /api/analytics/page-views", RequireAdmin(authenticator, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		series, err := service.PageViewSeries(request.Context(), analytics.MaximumSeriesDays)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "analytics_read_failed")
			return
		}
		response := make([]dailyPageViewResponse, 0, len(series))
		for _, item := range series {
			response = append(response, dailyPageViewResponse{Date: item.Day, Views: item.Views})
		}
		writeJSON(writer, http.StatusOK, pageViewResponse{Series: response})
	})))
}

func trackPublicPageViews(next http.Handler, authenticator Authenticator, service PageViewAnalytics) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		status := &responseStatusWriter{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(status, request)
		if request.Method != http.MethodGet || request.URL.Path != "/" || status.status < 200 || status.status >= 300 || obviousRobot(request.UserAgent()) {
			return
		}
		if authenticator != nil {
			err := authenticator.Authenticate(request.Context(), sessionToken(request))
			if err == nil {
				return
			}
			if !errors.Is(err, auth.ErrUnauthenticated) {
				slog.Warn("page-view administrator check failed", "error", err)
				return
			}
		}
		if err := service.RecordPageView(request.Context()); err != nil {
			slog.Warn("page-view recording failed", "error", err)
		}
	})
}

type responseStatusWriter struct {
	http.ResponseWriter
	status int
}

func (writer *responseStatusWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func obviousRobot(userAgent string) bool {
	lower := strings.ToLower(userAgent)
	for _, marker := range []string{
		"bot", "crawler", "spider", "slurp", "headless", "preview",
		"facebookexternalhit", "curl/", "wget/", "python-requests", "go-http-client", "uptimerobot",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
