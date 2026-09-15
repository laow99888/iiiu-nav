package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	DefaultSessionTTL = 30 * 24 * time.Hour
	sessionTokenBytes = 32
)

type Store interface {
	Admin(ctx context.Context) (Admin, bool, error)
	UpsertAdmin(ctx context.Context, passwordHash string) (Admin, error)
	ReplaceAdminPassword(ctx context.Context, passwordHash string, updatedAt time.Time) error
	CreateSession(ctx context.Context, session Session) error
	Session(ctx context.Context, tokenHash []byte) (Session, bool, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) (int64, error)
}

type Config struct {
	Store       Store
	Password    PasswordParams
	SessionTTL  time.Duration
	Random      io.Reader
	CurrentTime func() time.Time
}

type Manager struct {
	store       Store
	password    PasswordParams
	sessionTTL  time.Duration
	random      io.Reader
	currentTime func() time.Time
}

type SessionToken struct {
	Value     string
	ExpiresAt time.Time
}

func New(config Config) (*Manager, error) {
	if config.Store == nil {
		return nil, errors.New("authentication store is required")
	}
	if config.Password == (PasswordParams{}) {
		config.Password = DefaultPasswordParams()
	}
	if err := validatePasswordParams(config.Password); err != nil {
		return nil, err
	}
	if config.SessionTTL <= 0 {
		config.SessionTTL = DefaultSessionTTL
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	if config.CurrentTime == nil {
		config.CurrentTime = time.Now
	}
	return &Manager{
		store:       config.Store,
		password:    config.Password,
		sessionTTL:  config.SessionTTL,
		random:      config.Random,
		currentTime: config.CurrentTime,
	}, nil
}

func (manager *Manager) Bootstrapped(ctx context.Context) (bool, error) {
	_, found, err := manager.store.Admin(ctx)
	if err != nil {
		return false, fmt.Errorf("check administrator: %w", err)
	}
	return found, nil
}

func (manager *Manager) Login(ctx context.Context, password string) (SessionToken, error) {
	if err := manager.VerifyPassword(ctx, password); err != nil {
		return SessionToken{}, err
	}
	// Best-effort housekeeping: sessions that expired without ever being
	// presented again would otherwise linger until the next password change
	// or restore. A failed sweep does not block the login.
	_, _ = manager.store.DeleteExpiredSessions(ctx, manager.now())
	return manager.createSession(ctx)
}

func (manager *Manager) VerifyPassword(ctx context.Context, password string) error {
	if len(password) > maxPasswordBytes {
		return ErrInvalidCredentials
	}
	admin, found, err := manager.store.Admin(ctx)
	if err != nil {
		return fmt.Errorf("read administrator for verification: %w", err)
	}
	if !found {
		return ErrBootstrapRequired
	}
	valid, err := verifyPassword(password, admin.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify administrator password: %w", err)
	}
	if !valid {
		return ErrInvalidCredentials
	}
	return nil
}

func (manager *Manager) Authenticate(ctx context.Context, token string) error {
	tokenHash, ok := hashSessionToken(token)
	if !ok {
		return ErrUnauthenticated
	}
	session, found, err := manager.store.Session(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("read administrator session: %w", err)
	}
	if !found {
		return ErrUnauthenticated
	}
	if !session.ExpiresAt.After(manager.now()) {
		if err := manager.store.DeleteSession(ctx, tokenHash); err != nil {
			return fmt.Errorf("delete expired administrator session: %w", err)
		}
		return ErrUnauthenticated
	}
	return nil
}

func (manager *Manager) Logout(ctx context.Context, token string) error {
	tokenHash, ok := hashSessionToken(token)
	if !ok {
		return nil
	}
	if err := manager.store.DeleteSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("delete administrator session: %w", err)
	}
	return nil
}

func (manager *Manager) ChangePassword(ctx context.Context, token, currentPassword, newPassword string) error {
	if err := manager.Authenticate(ctx, token); err != nil {
		return err
	}
	if len(currentPassword) > maxPasswordBytes {
		return ErrInvalidCredentials
	}
	admin, found, err := manager.store.Admin(ctx)
	if err != nil {
		return fmt.Errorf("read administrator for password change: %w", err)
	}
	if !found {
		return ErrBootstrapRequired
	}
	valid, err := verifyPassword(currentPassword, admin.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify current administrator password: %w", err)
	}
	if !valid {
		return ErrInvalidCredentials
	}
	return manager.replacePassword(ctx, newPassword)
}

func (manager *Manager) ResetPassword(ctx context.Context, newPassword string) error {
	bootstrapped, err := manager.Bootstrapped(ctx)
	if err != nil {
		return err
	}
	if !bootstrapped {
		return ErrBootstrapRequired
	}
	return manager.replacePassword(ctx, newPassword)
}

func (manager *Manager) createSession(ctx context.Context) (SessionToken, error) {
	rawToken := make([]byte, sessionTokenBytes)
	if _, err := io.ReadFull(manager.random, rawToken); err != nil {
		return SessionToken{}, fmt.Errorf("generate administrator session: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	tokenHash := sha256.Sum256(rawToken)
	clear(rawToken)

	createdAt := manager.now()
	expiresAt := createdAt.Add(manager.sessionTTL)
	if err := manager.store.CreateSession(ctx, Session{
		TokenHash: tokenHash[:],
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}); err != nil {
		return SessionToken{}, fmt.Errorf("persist administrator session: %w", err)
	}
	return SessionToken{Value: token, ExpiresAt: expiresAt}, nil
}

func (manager *Manager) replacePassword(ctx context.Context, password string) error {
	encodedHash, err := hashPassword(password, manager.password, manager.random)
	if err != nil {
		return err
	}
	if err := manager.store.ReplaceAdminPassword(ctx, encodedHash, manager.now()); err != nil {
		return fmt.Errorf("replace administrator password: %w", err)
	}
	return nil
}

func (manager *Manager) now() time.Time {
	return manager.currentTime().UTC().Truncate(time.Millisecond)
}

func hashSessionToken(token string) ([]byte, bool) {
	rawToken, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(rawToken) != sessionTokenBytes {
		return nil, false
	}
	tokenHash := sha256.Sum256(rawToken)
	clear(rawToken)
	return tokenHash[:], true
}
