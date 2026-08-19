package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"iiiu-nav/internal/auth"
)

func (store *Store) UpsertAdmin(ctx context.Context, passwordHash string) (auth.Admin, error) {
	now := time.Now().UTC()
	if _, err := store.database.ExecContext(
		ctx,
		`INSERT INTO admin (id, password_hash, updated_at) VALUES (1, ?, ?)
         ON CONFLICT (id) DO UPDATE SET password_hash = excluded.password_hash, updated_at = excluded.updated_at`,
		passwordHash,
		timestamp(now),
	); err != nil {
		return auth.Admin{}, fmt.Errorf("upsert administrator: %w", err)
	}
	return auth.Admin{ID: 1, PasswordHash: passwordHash, UpdatedAt: now}, nil
}

func (store *Store) ReplaceAdminPassword(ctx context.Context, passwordHash string, updatedAt time.Time) error {
	transaction, err := store.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin administrator password replacement: %w", err)
	}
	defer transaction.Rollback()

	result, err := transaction.ExecContext(
		ctx,
		`UPDATE admin SET password_hash = ?, updated_at = ? WHERE id = 1`,
		passwordHash,
		timestamp(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("replace administrator password: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read replaced administrator count: %w", err)
	}
	if updated != 1 {
		return errors.New("administrator does not exist")
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return fmt.Errorf("invalidate administrator sessions: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit administrator password replacement: %w", err)
	}
	return nil
}

func (store *Store) Admin(ctx context.Context) (auth.Admin, bool, error) {
	var admin auth.Admin
	var updatedAt int64
	err := store.database.QueryRowContext(
		ctx,
		`SELECT id, password_hash, updated_at FROM admin WHERE id = 1`,
	).Scan(&admin.ID, &admin.PasswordHash, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Admin{}, false, nil
	}
	if err != nil {
		return auth.Admin{}, false, fmt.Errorf("read administrator: %w", err)
	}
	admin.UpdatedAt = timeFromTimestamp(updatedAt)
	return admin, true, nil
}

func (store *Store) CreateSession(ctx context.Context, session auth.Session) error {
	if _, err := store.database.ExecContext(
		ctx,
		`INSERT INTO sessions (token_hash, expires_at, created_at) VALUES (?, ?, ?)`,
		session.TokenHash,
		timestamp(session.ExpiresAt),
		timestamp(session.CreatedAt),
	); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (store *Store) Session(ctx context.Context, tokenHash []byte) (auth.Session, bool, error) {
	var session auth.Session
	var expiresAt int64
	var createdAt int64
	err := store.database.QueryRowContext(
		ctx,
		`SELECT token_hash, expires_at, created_at FROM sessions WHERE token_hash = ?`,
		tokenHash,
	).Scan(&session.TokenHash, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, false, nil
	}
	if err != nil {
		return auth.Session{}, false, fmt.Errorf("read session: %w", err)
	}
	session.ExpiresAt = timeFromTimestamp(expiresAt)
	session.CreatedAt = timeFromTimestamp(createdAt)
	return session, true, nil
}

func (store *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := store.database.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (store *Store) DeleteAllSessions(ctx context.Context) error {
	if _, err := store.database.ExecContext(ctx, `DELETE FROM sessions`); err != nil {
		return fmt.Errorf("delete all sessions: %w", err)
	}
	return nil
}

func (store *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) (int64, error) {
	result, err := store.database.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, timestamp(now))
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted session count: %w", err)
	}
	return deleted, nil
}
