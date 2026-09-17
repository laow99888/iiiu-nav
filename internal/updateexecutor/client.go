// Package updateexecutor is the application-side adapter for the optional
// host update executor: a tiny HTTP client over a shared unix socket. When
// no executor is configured, the application keeps the documented manual
// update path.
package updateexecutor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	ErrBusy           = errors.New("an update is already in progress")
	ErrInvalidVersion = errors.New("invalid target version")
	ErrNotConfigured  = errors.New("update executor is not configured")
)

// Status mirrors the executor state machine (see internal/hostexecutor).
type Status struct {
	State         string    `json:"state"`
	Phase         string    `json:"phase"`
	TargetVersion string    `json:"targetVersion,omitempty"`
	Message       string    `json:"message,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type StartRequest struct {
	Version       string `json:"version"`
	CurrentSchema int    `json:"currentSchema"`
	BackupName    string `json:"backupName"`
}

type Client struct {
	socketPath string
	client     *http.Client
}

// New returns a client for the executor socket, or nil when socketPath is
// empty so callers can treat "not configured" uniformly.
func New(socketPath string) *Client {
	if socketPath == "" {
		return nil
	}
	return &Client{
		socketPath: socketPath,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

// Available reports whether the executor socket is reachable right now.
func (client *Client) Available() bool {
	if client == nil {
		return false
	}
	_, err := os.Stat(client.socketPath)
	return err == nil
}

// Start asks the executor to begin the update; it returns once the executor
// accepted the request, progress is observed through Status.
func (client *Client) Start(ctx context.Context, request StartRequest) error {
	if client == nil {
		return ErrNotConfigured
	}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://executor/start", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	return client.do(httpRequest, func(response *http.Response) error {
		switch response.StatusCode {
		case http.StatusAccepted:
			return nil
		case http.StatusConflict:
			return ErrBusy
		case http.StatusUnprocessableEntity:
			return ErrInvalidVersion
		default:
			return fmt.Errorf("executor rejected start: HTTP %d", response.StatusCode)
		}
	})
}

func (client *Client) Status(ctx context.Context) (Status, error) {
	var status Status
	if client == nil {
		return status, ErrNotConfigured
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://executor/status", nil)
	if err != nil {
		return status, err
	}
	err = client.do(httpRequest, func(response *http.Response) error {
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("executor status: HTTP %d", response.StatusCode)
		}
		return json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&status)
	})
	return status, err
}

func (client *Client) do(request *http.Request, handle func(*http.Response) error) error {
	response, err := client.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return handle(response)
}
