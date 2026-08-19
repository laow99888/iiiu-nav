package server

import (
	"testing"
	"time"
)

func TestLoginLimiterWindowAndReset(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	limiter := newLoginLimiter(2, time.Minute, func() time.Time { return now })
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected first attempt to be allowed")
	}
	limiter.Failed("client")
	limiter.Failed("client")
	if allowed, retryAfter := limiter.Allow("client"); allowed || retryAfter != time.Minute {
		t.Fatalf("expected client to be blocked for a minute, allowed=%v retry=%s", allowed, retryAfter)
	}

	now = now.Add(time.Minute + time.Second)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected expired failures to be removed")
	}
	limiter.Failed("client")
	limiter.Reset("client")
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected reset client to be allowed")
	}
}
