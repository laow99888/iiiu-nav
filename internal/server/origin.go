package server

import (
	"net/http"
	"net/url"
	"strings"
)

func sameOriginOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if isSafeMethod(request.Method) {
			next.ServeHTTP(writer, request)
			return
		}

		setPrivateResponseHeaders(writer)
		writer.Header().Add("Vary", "Origin")
		writer.Header().Add("Vary", "Sec-Fetch-Site")
		fetchSite := strings.ToLower(strings.TrimSpace(request.Header.Get("Sec-Fetch-Site")))
		if fetchSite == "cross-site" || fetchSite == "same-site" {
			writeError(writer, http.StatusForbidden, "cross_origin_request")
			return
		}

		source := request.Header.Get("Origin")
		if source == "" {
			source = request.Header.Get("Referer")
		}
		if !matchesRequestOrigin(source, request) {
			writeError(writer, http.StatusForbidden, "origin_required")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func matchesRequestOrigin(source string, request *http.Request) bool {
	if source == "" || source == "null" {
		return false
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.User != nil || parsed.Host == "" {
		return false
	}
	expectedScheme := "http"
	if requestIsHTTPS(request) {
		expectedScheme = "https"
	}
	return strings.EqualFold(parsed.Scheme, expectedScheme) && strings.EqualFold(parsed.Host, request.Host)
}
