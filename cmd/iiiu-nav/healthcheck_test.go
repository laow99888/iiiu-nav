package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckHealth(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/healthz" {
			t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
		}
		_, _ = writer.Write([]byte("ok\n"))
	}))
	defer server.Close()
	if err := checkHealth(context.Background(), server.Client(), server.URL+"/healthz"); err != nil {
		t.Fatalf("expected healthy response: %v", err)
	}

	unhealthy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "unhealthy", http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()
	if err := checkHealth(context.Background(), unhealthy.Client(), unhealthy.URL); err == nil {
		t.Fatal("expected unhealthy response to fail")
	}
}
