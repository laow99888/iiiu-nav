package server

import (
	"context"
	"testing"
	"time"

	"iiiu-nav/internal/navigation"
)

type recordingLinks struct {
	LinkManager
	linkCalls int
	called    chan struct{}
}

func (links *recordingLinks) Link(ctx context.Context, _ int64) (navigation.Link, error) {
	links.linkCalls++
	links.called <- struct{}{}
	if ctx.Err() != nil {
		return navigation.Link{}, ctx.Err()
	}
	return navigation.Link{}, navigation.ErrLinkNotFound
}

func TestRefreshImportedDerivesContextFromBackground(t *testing.T) {
	background, cancel := context.WithCancel(context.Background())
	defer cancel()
	links := &recordingLinks{called: make(chan struct{}, 1)}
	handler := &metadataHandler{links: links, background: background}

	handler.refreshImported([]int64{1})
	select {
	case <-links.called:
	case <-time.After(time.Second):
		t.Fatal("expected the refresh to inspect the imported link")
	}
	if links.linkCalls != 1 {
		t.Fatalf("expected one link lookup, got %d", links.linkCalls)
	}
}

func TestRefreshImportedStopsAfterBackgroundCanceled(t *testing.T) {
	background, cancel := context.WithCancel(context.Background())
	cancel()
	links := &recordingLinks{called: make(chan struct{}, 1)}
	handler := &metadataHandler{links: links, background: background}

	handler.refreshImported([]int64{1})
	select {
	case <-links.called:
	case <-time.After(time.Second):
		t.Fatal("expected the lookup attempt even after cancellation")
	}
	if links.linkCalls != 1 {
		t.Fatalf("expected a single skipped lookup, got %d", links.linkCalls)
	}
}
