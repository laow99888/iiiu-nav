package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

func (manager *Manager) BootstrapFromFile(ctx context.Context, passwordFile string) (bool, error) {
	bootstrapped, err := manager.Bootstrapped(ctx)
	if err != nil {
		return false, err
	}
	if bootstrapped {
		return false, nil
	}
	if passwordFile == "" {
		return false, fmt.Errorf("%w: set ADMIN_PASSWORD_FILE to a readable password file", ErrBootstrapRequired)
	}

	password, err := readPasswordFile(passwordFile)
	if err != nil {
		return false, err
	}
	encodedHash, err := hashPassword(password, manager.password, manager.random)
	if err != nil {
		return false, fmt.Errorf("validate bootstrap password: %w", err)
	}
	if _, err := manager.store.UpsertAdmin(ctx, encodedHash); err != nil {
		return false, fmt.Errorf("create administrator: %w", err)
	}
	return true, nil
}

func ReadPasswordFile(passwordFile string) (string, error) {
	if passwordFile == "" {
		return "", errors.New("ADMIN_PASSWORD_FILE is required")
	}
	return readPasswordFile(passwordFile)
}

func readPasswordFile(passwordFile string) (string, error) {
	file, err := os.Open(passwordFile)
	if err != nil {
		return "", fmt.Errorf("open password file: %w", err)
	}
	defer file.Close()

	secret, err := io.ReadAll(io.LimitReader(file, maxPasswordBytes+3))
	if err != nil {
		return "", fmt.Errorf("read password file: %w", err)
	}
	defer clear(secret)
	secret = bytes.TrimRight(secret, "\r\n")
	if len(secret) > maxPasswordBytes {
		return "", ErrInvalidPassword
	}
	password := string(secret)
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	return password, nil
}
