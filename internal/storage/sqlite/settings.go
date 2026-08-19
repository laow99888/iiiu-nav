package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

func (store *Store) PutSetting(ctx context.Context, key string, value json.RawMessage) error {
	if key == "" {
		return errors.New("setting key is required")
	}
	if !json.Valid(value) {
		return errors.New("setting value must be valid JSON")
	}

	if _, err := store.database.ExecContext(
		ctx,
		`INSERT INTO settings (key, value_json) VALUES (?, ?)
         ON CONFLICT (key) DO UPDATE SET value_json = excluded.value_json`,
		key,
		string(value),
	); err != nil {
		return fmt.Errorf("put setting %q: %w", key, err)
	}
	return nil
}

func (store *Store) Setting(ctx context.Context, key string) (json.RawMessage, bool, error) {
	var value string
	err := store.database.QueryRowContext(
		ctx,
		`SELECT value_json FROM settings WHERE key = ?`,
		key,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read setting %q: %w", key, err)
	}
	return json.RawMessage(value), true, nil
}
