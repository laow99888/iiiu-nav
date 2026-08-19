package server

import (
	"context"
	"net/http"

	"iiiu-nav/internal/updatecheck"
)

type UpdateChecker interface {
	Check(context.Context, bool) updatecheck.Result
}

type updateStatusResponse struct {
	updatecheck.Result
	AutomaticUpdate bool `json:"automaticUpdate"`
}

func registerUpdateRoutes(mux *http.ServeMux, authenticator Authenticator, checker UpdateChecker) {
	handler := func(force bool) http.Handler {
		return RequireAdmin(authenticator, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			result := checker.Check(request.Context(), force)
			writeJSON(writer, http.StatusOK, updateStatusResponse{Result: result, AutomaticUpdate: false})
		}))
	}
	mux.Handle("GET /api/updates", handler(false))
	mux.Handle("POST /api/updates/check", handler(true))
}
