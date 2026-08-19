package auth

import "errors"

var (
	ErrBootstrapRequired  = errors.New("administrator bootstrap is required")
	ErrInvalidCredentials = errors.New("invalid administrator credentials")
	ErrUnauthenticated    = errors.New("administrator authentication is required")
	ErrInvalidPassword    = errors.New("password must contain at least 12 characters and at most 1024 bytes")
)
