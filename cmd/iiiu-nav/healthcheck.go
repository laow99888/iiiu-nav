package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

func healthcheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return checkHealth(ctx, http.DefaultClient, "http://127.0.0.1:8080/healthz")
}

func checkHealth(ctx context.Context, client *http.Client, target string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 16))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK || string(body) != "ok\n" {
		return fmt.Errorf("health endpoint returned status %d", response.StatusCode)
	}
	return nil
}
