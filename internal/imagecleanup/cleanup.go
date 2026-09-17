// Package imagecleanup owns the reference-aware retirement rule for stored
// images: a stored file may be deleted only when no active record references
// it, and cleanup failures are logged instead of silently dropped.
package imagecleanup

import (
	"context"
	"log/slog"

	"iiiu-nav/internal/imagestore"
)

// ReferenceCounter reports how many active records reference an image path.
type ReferenceCounter interface {
	LogoReferenceCount(ctx context.Context, publicPath string) (int, error)
}

// Retire removes each path from store when counter reports zero references.
// Empty and foreign paths are ignored, and counting or removal failures are
// logged without failing the caller: cleanup is best-effort by design.
func Retire(ctx context.Context, logger *slog.Logger, store *imagestore.Store, counter ReferenceCounter, paths ...string) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if store == nil || counter == nil {
		return
	}
	for _, publicPath := range paths {
		if publicPath == "" {
			continue
		}
		if _, owned := store.Filename(publicPath); !owned {
			continue
		}
		count, err := counter.LogoReferenceCount(ctx, publicPath)
		if err != nil {
			logger.Warn("image cleanup skipped", "path", publicPath, "error", err)
			continue
		}
		if count > 0 {
			continue
		}
		if err := store.Remove(publicPath); err != nil {
			logger.Warn("image cleanup failed", "path", publicPath, "error", err)
		}
	}
}
