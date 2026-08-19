package server

import (
	"net/http"
	"sync"
)

type maintenanceGate struct {
	sync.RWMutex
}

func maintenanceRequests(gate *maintenanceGate, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.URL.Path == "/api/restore" {
			next.ServeHTTP(writer, request)
			return
		}
		gate.RLock()
		defer gate.RUnlock()
		next.ServeHTTP(writer, request)
	})
}
