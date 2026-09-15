package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthTargetFollowsConfiguredAddress(t *testing.T) {
	t.Parallel()
	cases := []struct {
		address string
		want    string
	}{
		{address: ":8080", want: "http://127.0.0.1:8080/healthz"},
		{address: "0.0.0.0:9000", want: "http://127.0.0.1:9000/healthz"},
		{address: "192.168.1.4:8080", want: "http://192.168.1.4:8080/healthz"},
		{address: "[::1]:8080", want: "http://[::1]:8080/healthz"},
	}
	for _, testCase := range cases {
		got, err := healthTarget(testCase.address)
		if err != nil {
			t.Fatalf("healthTarget(%q): %v", testCase.address, err)
		}
		if got != testCase.want {
			t.Fatalf("healthTarget(%q) = %q, want %q", testCase.address, got, testCase.want)
		}
	}
	if _, err := healthTarget("missing-port"); err == nil {
		t.Fatal("expected an address without a port to fail")
	}
}

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
