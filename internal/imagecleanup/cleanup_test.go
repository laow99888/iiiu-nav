package imagecleanup

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"iiiu-nav/internal/imagestore"
)

type counterFunc func(ctx context.Context, publicPath string) (int, error)

func (counter counterFunc) LogoReferenceCount(ctx context.Context, publicPath string) (int, error) {
	return counter(ctx, publicPath)
}

func newTestStore(t *testing.T) (*imagestore.Store, string) {
	t.Helper()
	root := t.TempDir()
	store := imagestore.NewLogos(root)
	sprite := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	sprite.Set(0, 0, color.NRGBA{R: 240, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, sprite); err != nil {
		t.Fatalf("encode fixture image: %v", err)
	}
	publicPath, err := store.Save(&buffer)
	if err != nil {
		t.Fatalf("save fixture image: %v", err)
	}
	return store, publicPath
}

var logBuffer bytes.Buffer

func newTestLogger() *slog.Logger {
	logBuffer.Reset()
	return slog.New(slog.NewTextHandler(&logBuffer, nil))
}

func loggedWarnings() []string {
	var warnings []string
	for _, line := range strings.Split(strings.TrimSpace(logBuffer.String()), "\n") {
		if line != "" {
			warnings = append(warnings, line)
		}
	}
	return warnings
}

func assertStored(t *testing.T, store *imagestore.Store, publicPath string, wantStored bool) {
	t.Helper()
	name, owned := store.Filename(publicPath)
	if !owned {
		t.Fatalf("store rejects fixture path %q", publicPath)
	}
	_, err := os.Stat(filepath.Join(store.Root(), name))
	if wantStored && err != nil {
		t.Fatalf("expected %q to remain stored: %v", publicPath, err)
	}
	if !wantStored && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be removed, stat err=%v", publicPath, err)
	}
}

func TestRetireRemovesUnreferencedImage(t *testing.T) {
	store, publicPath := newTestStore(t)
	logger := newTestLogger()

	Retire(context.Background(), logger, store, counterFunc(func(context.Context, string) (int, error) {
		return 0, nil
	}), publicPath)

	assertStored(t, store, publicPath, false)
	if warnings := loggedWarnings(); len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
}

func TestRetireKeepsReferencedImage(t *testing.T) {
	store, publicPath := newTestStore(t)
	logger := newTestLogger()

	Retire(context.Background(), logger, store, counterFunc(func(context.Context, string) (int, error) {
		return 2, nil
	}), publicPath)

	assertStored(t, store, publicPath, true)
}

func TestRetireLogsCountingFailureAndKeepsImage(t *testing.T) {
	store, publicPath := newTestStore(t)
	logger := newTestLogger()

	Retire(context.Background(), logger, store, counterFunc(func(context.Context, string) (int, error) {
		return 0, errors.New("database busy")
	}), publicPath)

	assertStored(t, store, publicPath, true)
	warnings := loggedWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "image cleanup skipped") || !strings.Contains(warnings[0], "database busy") {
		t.Fatalf("expected one skip warning, got %v", warnings)
	}
}

func TestRetireLogsRemovalFailure(t *testing.T) {
	store, publicPath := newTestStore(t)
	name, _ := store.Filename(publicPath)
	// Replace the stored file with a non-empty directory so removal fails on
	// every supported platform.
	if err := os.Remove(filepath.Join(store.Root(), name)); err != nil {
		t.Fatalf("prepare removal failure: %v", err)
	}
	if err := os.Mkdir(filepath.Join(store.Root(), name), 0o755); err != nil {
		t.Fatalf("prepare removal failure: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Root(), name, "keep.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("prepare removal failure: %v", err)
	}
	logger := newTestLogger()

	Retire(context.Background(), logger, store, counterFunc(func(context.Context, string) (int, error) {
		return 0, nil
	}), publicPath)

	warnings := loggedWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "image cleanup failed") {
		t.Fatalf("expected one failure warning, got %v", warnings)
	}
}

func TestRetireIgnoresEmptyAndForeignPaths(t *testing.T) {
	store, _ := newTestStore(t)
	logger := newTestLogger()
	var counted []string
	counter := counterFunc(func(_ context.Context, publicPath string) (int, error) {
		counted = append(counted, publicPath)
		return 0, nil
	})

	Retire(context.Background(), logger, store, counter, "", "/etc/passwd", imagestore.BackgroundPrefix+"escape.png")

	if len(counted) != 0 {
		t.Fatalf("counter must not be called for foreign paths, got %v", counted)
	}
}

func TestRetireToleratesNilStoreAndCounter(t *testing.T) {
	store, publicPath := newTestStore(t)
	logger := newTestLogger()

	Retire(context.Background(), logger, nil, counterFunc(func(context.Context, string) (int, error) {
		return 0, nil
	}), publicPath)
	Retire(context.Background(), logger, store, nil, publicPath)

	assertStored(t, store, publicPath, true)
}
