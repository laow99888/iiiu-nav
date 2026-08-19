package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"iiiu-nav/internal/auth"
)

func TestSettingsRepositoryStoresValidJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)

	if _, found, err := store.Setting(ctx, "site"); err != nil || found {
		t.Fatalf("expected missing setting, found=%v err=%v", found, err)
	}
	if err := store.PutSetting(ctx, "site", json.RawMessage(`{"name":"iiiu-nav"}`)); err != nil {
		t.Fatalf("put setting: %v", err)
	}
	if err := store.PutSetting(ctx, "site", json.RawMessage(`{"name":"My Nav"}`)); err != nil {
		t.Fatalf("replace setting: %v", err)
	}

	value, found, err := store.Setting(ctx, "site")
	if err != nil {
		t.Fatalf("read setting: %v", err)
	}
	if !found || string(value) != `{"name":"My Nav"}` {
		t.Fatalf("unexpected setting: found=%v value=%s", found, value)
	}
	if err := store.PutSetting(ctx, "invalid", json.RawMessage(`{"name":`)); err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}

func TestAuthRepositoryStoresAdminAndSessions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)

	if _, found, err := store.Admin(ctx); err != nil || found {
		t.Fatalf("expected missing administrator, found=%v err=%v", found, err)
	}
	if _, err := store.UpsertAdmin(ctx, "$argon2id$first"); err != nil {
		t.Fatalf("create administrator: %v", err)
	}
	if _, err := store.UpsertAdmin(ctx, "$argon2id$second"); err != nil {
		t.Fatalf("replace administrator: %v", err)
	}
	admin, found, err := store.Admin(ctx)
	if err != nil {
		t.Fatalf("read administrator: %v", err)
	}
	if !found || admin.ID != 1 || admin.PasswordHash != "$argon2id$second" {
		t.Fatalf("unexpected administrator: found=%v admin=%+v", found, admin)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	expiredHash := bytes.Repeat([]byte{1}, 32)
	activeHash := bytes.Repeat([]byte{2}, 32)
	for _, session := range []auth.Session{
		{TokenHash: expiredHash, CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour)},
		{TokenHash: activeHash, CreatedAt: now, ExpiresAt: now.Add(time.Hour)},
	} {
		if err := store.CreateSession(ctx, session); err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	deleted, err := store.DeleteExpiredSessions(ctx, now)
	if err != nil {
		t.Fatalf("delete expired sessions: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one expired session, deleted %d", deleted)
	}
	if _, found, err := store.Session(ctx, expiredHash); err != nil || found {
		t.Fatalf("expected expired session to be absent, found=%v err=%v", found, err)
	}
	active, found, err := store.Session(ctx, activeHash)
	if err != nil {
		t.Fatalf("read active session: %v", err)
	}
	if !found || !bytes.Equal(active.TokenHash, activeHash) || !active.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("unexpected active session: found=%v session=%+v", found, active)
	}
	if err := store.DeleteAllSessions(ctx); err != nil {
		t.Fatalf("delete all sessions: %v", err)
	}
	if _, found, err := store.Session(ctx, activeHash); err != nil || found {
		t.Fatalf("expected active session to be deleted, found=%v err=%v", found, err)
	}
}

func TestReplaceAdminPasswordIsAtomic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := openTestStore(t)
	if _, err := store.UpsertAdmin(ctx, "$argon2id$old"); err != nil {
		t.Fatalf("create administrator: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	tokenHash := bytes.Repeat([]byte{3}, 32)
	if err := store.CreateSession(ctx, auth.Session{
		TokenHash: tokenHash,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	updatedAt := now.Add(time.Minute)
	if err := store.ReplaceAdminPassword(ctx, "$argon2id$new", updatedAt); err != nil {
		t.Fatalf("replace administrator password: %v", err)
	}
	admin, found, err := store.Admin(ctx)
	if err != nil || !found {
		t.Fatalf("read administrator: found=%v err=%v", found, err)
	}
	if admin.PasswordHash != "$argon2id$new" || !admin.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected administrator after replacement: %+v", admin)
	}
	if _, found, err := store.Session(ctx, tokenHash); err != nil || found {
		t.Fatalf("expected replacement to invalidate session: found=%v err=%v", found, err)
	}
}
