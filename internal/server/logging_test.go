package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogServerErrorsRecordsOnlyFailedRequests(t *testing.T) {
	t.Parallel()
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	handler := logServerErrors(logger, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/boom" {
			writeError(writer, http.StatusInternalServerError, "unexpected_failure")
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	}))

	failed := httptest.NewRecorder()
	handler.ServeHTTP(failed, httptest.NewRequest(http.MethodGet, "/boom", nil))
	logs := buffer.String()
	if !strings.Contains(logs, `"path":"/boom"`) || !strings.Contains(logs, `"status":500`) {
		t.Fatalf("expected a warning for the failed request, got %s", logs)
	}

	buffer.Reset()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil))
	if buffer.Len() != 0 {
		t.Fatalf("expected no log for a 404 response, got %s", buffer.String())
	}
}

func TestStatusRecorderUnwrapsForResponseController(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		record := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		controller := http.NewResponseController(record)
		if err := controller.SetWriteDeadline(time.Time{}); err != nil {
			http.Error(writer, "deadline control not supported", http.StatusInternalServerError)
			return
		}
		record.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	response, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("request deadline-control handler: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected deadline control through the recorder wrapper, got status %d", response.StatusCode)
	}
}
