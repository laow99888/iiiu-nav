package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

func healthcheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	target, err := healthTarget(environment("IIU_NAV_ADDR", ":8080"))
	if err != nil {
		return err
	}
	return checkHealth(ctx, http.DefaultClient, target)
}

func healthTarget(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("parse health check address %q: %w", address, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz", nil
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
