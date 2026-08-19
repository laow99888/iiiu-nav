package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBootstrapReadsPasswordFileOnlyWhenRequired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	manager := newTestManager(t, store, func() time.Time { return now })
	passwordFile := filepath.Join(t.TempDir(), "admin-password")
	if err := os.WriteFile(passwordFile, []byte("a secure bootstrap password\r\n"), 0o600); err != nil {
		t.Fatalf("write password file: %v", err)
	}

	created, err := manager.BootstrapFromFile(ctx, passwordFile)
	if err != nil || !created {
		t.Fatalf("bootstrap administrator: created=%v err=%v", created, err)
	}
	if store.admin.PasswordHash == "a secure bootstrap password" {
		t.Fatal("password was stored as plaintext")
	}
	valid, err := verifyPassword("a secure bootstrap password", store.admin.PasswordHash)
	if err != nil || !valid {
		t.Fatalf("verify stored password: valid=%v err=%v", valid, err)
	}

	created, err = manager.BootstrapFromFile(ctx, filepath.Join(t.TempDir(), "missing"))
	if err != nil || created {
		t.Fatalf("existing administrator should skip password file: created=%v err=%v", created, err)
	}
}

func TestBootstrapRequiresPasswordFile(t *testing.T) {
	t.Parallel()

	manager := newTestManager(t, newMemoryStore(), time.Now)
	if _, err := manager.BootstrapFromFile(context.Background(), ""); !errors.Is(err, ErrBootstrapRequired) {
		t.Fatalf("expected bootstrap requirement, got %v", err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	manager := newTestManager(t, store, func() time.Time { return now })
	bootstrapMemoryAdmin(t, manager, store, "correct horse battery staple")

	if _, err := manager.Login(ctx, "incorrect password value"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	token, err := manager.Login(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token.Value == "" || len(store.sessions) != 1 {
		t.Fatalf("expected one hashed session, token=%q sessions=%d", token.Value, len(store.sessions))
	}
	if err := manager.Authenticate(ctx, token.Value); err != nil {
		t.Fatalf("authenticate session: %v", err)
	}
	if err := manager.Logout(ctx, token.Value); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if err := manager.Authenticate(ctx, token.Value); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected logged-out session rejection, got %v", err)
	}
	token, err = manager.Login(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatalf("login before expiry check: %v", err)
	}

	now = now.Add(DefaultSessionTTL + time.Second)
	if err := manager.Authenticate(ctx, token.Value); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected expired session rejection, got %v", err)
	}
	if len(store.sessions) != 0 {
		t.Fatal("expired session was not deleted")
	}
}

func TestChangeAndResetPasswordInvalidateSessions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	store := newMemoryStore()
	manager := newTestManager(t, store, func() time.Time { return now })
	bootstrapMemoryAdmin(t, manager, store, "original administrator password")

	first, err := manager.Login(ctx, "original administrator password")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	if _, err := manager.Login(ctx, "original administrator password"); err != nil {
		t.Fatalf("second login: %v", err)
	}
	if err := manager.ChangePassword(ctx, first.Value, "incorrect current password", "replacement administrator password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected current password rejection, got %v", err)
	}
	if len(store.sessions) != 2 {
		t.Fatal("failed password change invalidated sessions")
	}
	if err := manager.ChangePassword(ctx, first.Value, "original administrator password", "replacement administrator password"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if len(store.sessions) != 0 {
		t.Fatal("password change did not invalidate every session")
	}
	if _, err := manager.Login(ctx, "original administrator password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password still works: %v", err)
	}
	if _, err := manager.Login(ctx, "replacement administrator password"); err != nil {
		t.Fatalf("replacement password login: %v", err)
	}
	if err := manager.ResetPassword(ctx, "recovered administrator password"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if len(store.sessions) != 0 {
		t.Fatal("password reset did not invalidate every session")
	}
}

func newTestManager(t *testing.T, store Store, now func() time.Time) *Manager {
	t.Helper()
	manager, err := New(Config{
		Store:       store,
		Password:    testPasswordParams,
		Random:      &sequenceReader{},
		CurrentTime: now,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}
	return manager
}

func bootstrapMemoryAdmin(t *testing.T, manager *Manager, store *memoryStore, password string) {
	t.Helper()
	encoded, err := hashPassword(password, manager.password, manager.random)
	if err != nil {
		t.Fatalf("hash bootstrap password: %v", err)
	}
	store.admin = Admin{ID: 1, PasswordHash: encoded, UpdatedAt: manager.now()}
	store.hasAdmin = true
}

type memoryStore struct {
	admin    Admin
	hasAdmin bool
	sessions map[string]Session
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sessions: make(map[string]Session)}
}

func (store *memoryStore) Admin(context.Context) (Admin, bool, error) {
	return store.admin, store.hasAdmin, nil
}

func (store *memoryStore) UpsertAdmin(_ context.Context, passwordHash string) (Admin, error) {
	store.admin = Admin{ID: 1, PasswordHash: passwordHash}
	store.hasAdmin = true
	return store.admin, nil
}

func (store *memoryStore) ReplaceAdminPassword(_ context.Context, passwordHash string, updatedAt time.Time) error {
	store.admin.PasswordHash = passwordHash
	store.admin.UpdatedAt = updatedAt
	store.sessions = make(map[string]Session)
	return nil
}

func (store *memoryStore) CreateSession(_ context.Context, session Session) error {
	store.sessions[string(session.TokenHash)] = session
	return nil
}

func (store *memoryStore) Session(_ context.Context, tokenHash []byte) (Session, bool, error) {
	session, found := store.sessions[string(tokenHash)]
	return session, found, nil
}

func (store *memoryStore) DeleteSession(_ context.Context, tokenHash []byte) error {
	delete(store.sessions, string(tokenHash))
	return nil
}

type sequenceReader struct {
	next byte
}

func (reader *sequenceReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		reader.next++
		buffer[index] = reader.next
	}
	return len(buffer), nil
}
