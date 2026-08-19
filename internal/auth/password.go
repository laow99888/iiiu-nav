package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltLength = 16
	passwordKeyLength  = 32
	minPasswordRunes   = 9
	maxPasswordBytes   = 1024
)

type PasswordParams struct {
	Memory  uint32
	Time    uint32
	Threads uint8
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{Memory: 64 * 1024, Time: 3, Threads: 4}
}

func ValidatePassword(password string) error {
	if len(password) > maxPasswordBytes || utf8.RuneCountInString(password) < minPasswordRunes {
		return ErrInvalidPassword
	}
	return nil
}

func hashPassword(password string, params PasswordParams, random io.Reader) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	if err := validatePasswordParams(params); err != nil {
		return "", err
	}

	salt := make([]byte, passwordSaltLength)
	if _, err := io.ReadFull(random, salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, passwordKeyLength)
	defer clear(key)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Time,
		params.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func verifyPassword(password, encodedHash string) (bool, error) {
	params, salt, expectedKey, err := parsePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}
	defer clear(expectedKey)

	key := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, uint32(len(expectedKey)))
	defer clear(key)
	return subtle.ConstantTimeCompare(key, expectedKey) == 1, nil
}

func parsePasswordHash(encodedHash string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id hash format")
	}

	version, err := strconv.Atoi(strings.TrimPrefix(parts[2], "v="))
	if err != nil || version != argon2.Version || parts[2] != fmt.Sprintf("v=%d", version) {
		return PasswordParams{}, nil, nil, errors.New("unsupported Argon2id hash version")
	}
	params, err := parsePasswordParams(parts[3])
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id salt")
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) < 16 || len(key) > 64 {
		return PasswordParams{}, nil, nil, errors.New("invalid Argon2id key")
	}
	return params, salt, key, nil
}

func parsePasswordParams(encoded string) (PasswordParams, error) {
	parts := strings.Split(encoded, ",")
	if len(parts) != 3 {
		return PasswordParams{}, errors.New("invalid Argon2id parameters")
	}

	memory, err := parseUintParameter(parts[0], "m", 32)
	if err != nil {
		return PasswordParams{}, err
	}
	timeCost, err := parseUintParameter(parts[1], "t", 32)
	if err != nil {
		return PasswordParams{}, err
	}
	threads, err := parseUintParameter(parts[2], "p", 8)
	if err != nil {
		return PasswordParams{}, err
	}
	params := PasswordParams{Memory: uint32(memory), Time: uint32(timeCost), Threads: uint8(threads)}
	if err := validatePasswordParams(params); err != nil {
		return PasswordParams{}, err
	}
	return params, nil
}

func parseUintParameter(encoded, name string, bitSize int) (uint64, error) {
	prefix := name + "="
	if !strings.HasPrefix(encoded, prefix) {
		return 0, fmt.Errorf("missing Argon2id parameter %s", name)
	}
	value, err := strconv.ParseUint(strings.TrimPrefix(encoded, prefix), 10, bitSize)
	if err != nil {
		return 0, fmt.Errorf("parse Argon2id parameter %s: %w", name, err)
	}
	return value, nil
}

func validatePasswordParams(params PasswordParams) error {
	if params.Memory < 7*1024 || params.Memory > 256*1024 {
		return errors.New("Argon2id memory is outside the supported range")
	}
	if params.Time < 1 || params.Time > 10 {
		return errors.New("Argon2id time is outside the supported range")
	}
	if params.Threads < 1 || params.Threads > 8 {
		return errors.New("Argon2id threads are outside the supported range")
	}
	return nil
}
