package server

import (
	"sync"
	"time"
)

type loginLimiter struct {
	mutex       sync.Mutex
	maxFailures int
	window      time.Duration
	currentTime func() time.Time
	failures    map[string][]time.Time
}

func newLoginLimiter(maxFailures int, window time.Duration, currentTime func() time.Time) *loginLimiter {
	return &loginLimiter{
		maxFailures: maxFailures,
		window:      window,
		currentTime: currentTime,
		failures:    make(map[string][]time.Time),
	}
}

func (limiter *loginLimiter) Allow(key string) (bool, time.Duration) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	now := limiter.currentTime()
	recent := limiter.recentFailures(key, now)
	if len(recent) < limiter.maxFailures {
		return true, 0
	}
	retryAfter := limiter.window - now.Sub(recent[0])
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	return false, retryAfter
}

func (limiter *loginLimiter) Failed(key string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	now := limiter.currentTime()
	recent := limiter.recentFailures(key, now)
	limiter.failures[key] = append(recent, now)
}

func (limiter *loginLimiter) Reset(key string) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	delete(limiter.failures, key)
}

func (limiter *loginLimiter) recentFailures(key string, now time.Time) []time.Time {
	failures := limiter.failures[key]
	cutoff := now.Add(-limiter.window)
	firstRecent := 0
	for firstRecent < len(failures) && !failures[firstRecent].After(cutoff) {
		firstRecent++
	}
	if firstRecent == len(failures) {
		delete(limiter.failures, key)
		return nil
	}
	recent := failures[firstRecent:]
	limiter.failures[key] = recent
	return recent
}
