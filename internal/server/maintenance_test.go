package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMaintenanceGateKeepsHealthAvailableDuringRestore(t *testing.T) {
	t.Parallel()
	gate := &maintenanceGate{}
	gate.Lock()
	handler := maintenanceRequests(gate, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusNoContent {
		t.Fatalf("expected healthz to bypass the maintenance gate, got %d", health.Code)
	}
	apiHealth := httptest.NewRecorder()
	handler.ServeHTTP(apiHealth, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if apiHealth.Code != http.StatusNoContent {
		t.Fatalf("expected api health to bypass the maintenance gate, got %d", apiHealth.Code)
	}

	blocked := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/navigation", nil))
		close(blocked)
	}()
	select {
	case <-blocked:
		t.Fatal("expected a read to wait for the maintenance gate")
	case <-time.After(100 * time.Millisecond):
	}
	gate.Unlock()
	select {
	case <-blocked:
	case <-time.After(time.Second):
		t.Fatal("expected the blocked read to finish after the gate was released")
	}
}
