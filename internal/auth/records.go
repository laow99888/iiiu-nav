package auth

import "time"

type Admin struct {
	ID           int64
	PasswordHash string
	UpdatedAt    time.Time
}

type Session struct {
	TokenHash []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}
