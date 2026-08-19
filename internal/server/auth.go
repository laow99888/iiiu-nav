package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"iiiu-nav/internal/auth"
)

const SessionCookieName = "iiiu_nav_session"

type Authenticator interface {
	Login(ctx context.Context, password string) (auth.SessionToken, error)
	Authenticate(ctx context.Context, token string) error
	Logout(ctx context.Context, token string) error
	ChangePassword(ctx context.Context, token, currentPassword, newPassword string) error
}

type authHandler struct {
	authenticator Authenticator
	limiter       *loginLimiter
}

type loginRequest struct {
	Password string `json:"password"`
}

type passwordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func newAuthHandler(authenticator Authenticator) *authHandler {
	return &authHandler{
		authenticator: authenticator,
		limiter:       newLoginLimiter(5, 10*time.Minute, time.Now),
	}
}

func registerAuthRoutes(mux *http.ServeMux, handler *authHandler) {
	mux.HandleFunc("POST /api/auth/login", handler.login)
	mux.HandleFunc("POST /api/auth/logout", handler.logout)
	mux.HandleFunc("GET /api/auth/session", handler.session)
	mux.HandleFunc("POST /api/auth/password", handler.changePassword)
}

func (handler *authHandler) login(writer http.ResponseWriter, request *http.Request) {
	setPrivateResponseHeaders(writer)
	client := clientAddress(request)
	if allowed, retryAfter := handler.limiter.Allow(client); !allowed {
		writer.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Round(time.Second).Seconds())))
		writeError(writer, http.StatusTooManyRequests, "login_rate_limited")
		return
	}

	var body loginRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	session, err := handler.authenticator.Login(request.Context(), body.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrBootstrapRequired) {
		handler.limiter.Failed(client)
		writeError(writer, http.StatusUnauthorized, "invalid_credentials")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "authentication_failed")
		return
	}

	handler.limiter.Reset(client)
	setSessionCookie(writer, request, session)
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *authHandler) logout(writer http.ResponseWriter, request *http.Request) {
	setPrivateResponseHeaders(writer)
	token := sessionToken(request)
	if err := handler.authenticator.Logout(request.Context(), token); err != nil {
		writeError(writer, http.StatusInternalServerError, "logout_failed")
		return
	}
	clearSessionCookie(writer, request)
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *authHandler) session(writer http.ResponseWriter, request *http.Request) {
	setPrivateResponseHeaders(writer)
	token := sessionToken(request)
	if token == "" {
		writeJSON(writer, http.StatusOK, map[string]bool{"authenticated": false})
		return
	}
	if err := handler.authenticator.Authenticate(request.Context(), token); errors.Is(err, auth.ErrUnauthenticated) {
		clearSessionCookie(writer, request)
		writeJSON(writer, http.StatusOK, map[string]bool{"authenticated": false})
		return
	} else if err != nil {
		writeError(writer, http.StatusInternalServerError, "authentication_failed")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"authenticated": true})
}

func (handler *authHandler) changePassword(writer http.ResponseWriter, request *http.Request) {
	setPrivateResponseHeaders(writer)
	var body passwordRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	token := sessionToken(request)
	err := handler.authenticator.ChangePassword(request.Context(), token, body.CurrentPassword, body.NewPassword)
	switch {
	case errors.Is(err, auth.ErrUnauthenticated):
		clearSessionCookie(writer, request)
		writeError(writer, http.StatusUnauthorized, "authentication_required")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(writer, http.StatusUnauthorized, "current_password_invalid")
	case errors.Is(err, auth.ErrInvalidPassword):
		writeError(writer, http.StatusUnprocessableEntity, "invalid_password")
	case err != nil:
		writeError(writer, http.StatusInternalServerError, "password_change_failed")
	default:
		clearSessionCookie(writer, request)
		writer.WriteHeader(http.StatusNoContent)
	}
}

func RequireAdmin(authenticator Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		setPrivateResponseHeaders(writer)
		if err := authenticator.Authenticate(request.Context(), sessionToken(request)); err != nil {
			if errors.Is(err, auth.ErrUnauthenticated) {
				clearSessionCookie(writer, request)
				writeError(writer, http.StatusUnauthorized, "authentication_required")
				return
			}
			writeError(writer, http.StatusInternalServerError, "authentication_failed")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func setSessionCookie(writer http.ResponseWriter, request *http.Request, session auth.SessionToken) {
	http.SetCookie(writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.Value,
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   max(1, int(time.Until(session.ExpiresAt).Seconds())),
		HttpOnly: true,
		Secure:   requestIsHTTPS(request),
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookie(writer http.ResponseWriter, request *http.Request) {
	http.SetCookie(writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   requestIsHTTPS(request),
		SameSite: http.SameSiteStrictMode,
	})
}

func sessionToken(request *http.Request) string {
	cookie, err := request.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func clientAddress(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}

func requestIsHTTPS(request *http.Request) bool {
	if request.TLS != nil {
		return true
	}
	forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwarded, "https")
}

func setPrivateResponseHeaders(writer http.ResponseWriter) {
	writer.Header().Set("Cache-Control", "no-store")
}
