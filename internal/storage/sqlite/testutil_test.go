package sqlite

import (
	"context"
	"path/filepath"
)

type testStoreTB interface {
	Helper()
	TempDir() string
	Cleanup(func())
	Fatalf(string, ...any)
	Errorf(string, ...any)
}

func openTestStore(t testStoreTB) *Store {
	t.Helper()

	databasePath := filepath.Join(t.TempDir(), "nav.db")
	store, err := Open(context.Background(), databasePath)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close test store: %v", err)
		}
	})
	return store
}
