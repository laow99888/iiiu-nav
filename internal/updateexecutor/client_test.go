package updateexecutor

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// startFakeExecutor serves the executor protocol on a real unix socket and
// records the last start request it accepted.
func startFakeExecutor(t *testing.T, status int, body string) (*Client, *[]StartRequest) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "executor.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	var started []StartRequest
	mux := http.NewServeMux()
	mux.HandleFunc("POST /start", func(writer http.ResponseWriter, request *http.Request) {
		var request_ StartRequest
		if err := json.NewDecoder(request.Body).Decode(&request_); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		started = append(started, request_)
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	})
	mux.HandleFunc("GET /status", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"state":"running","phase":"pulling","targetVersion":"v1.1.0"}`))
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = os.Remove(socketPath)
	})
	return New(socketPath), &started
}

func TestClientTalksToExecutorSocket(t *testing.T) {
	client, started := startFakeExecutor(t, http.StatusAccepted, `{"accepted":true}`)
	if !client.Available() {
		t.Fatal("executor socket must be available")
	}
	if err := client.Start(context.Background(), StartRequest{Version: "v1.1.0", CurrentSchema: 3, BackupName: "pre.zip"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(*started) != 1 || (*started)[0].Version != "v1.1.0" || (*started)[0].CurrentSchema != 3 {
		t.Fatalf("unexpected start request: %+v", (*started)[0])
	}

	status, err := client.Status(context.Background())
	if err != nil || status.State != "running" || status.Phase != "pulling" {
		t.Fatalf("status: %+v %v", status, err)
	}
}

func TestClientMapsExecutorRejections(t *testing.T) {
	client, _ := startFakeExecutor(t, http.StatusConflict, `{"error":"update_already_running"}`)
	if err := client.Start(context.Background(), StartRequest{Version: "v1.1.0"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("start = %v, want ErrBusy", err)
	}
}

func TestNilClientIsNeverAvailable(t *testing.T) {
	var client *Client
	if client.Available() {
		t.Fatal("nil client must not be available")
	}
	if err := client.Start(context.Background(), StartRequest{Version: "v1.1.0"}); err == nil {
		t.Fatal("nil client start must fail")
	}
	if _, err := client.Status(context.Background()); err == nil {
		t.Fatal("nil client status must fail")
	}
}
