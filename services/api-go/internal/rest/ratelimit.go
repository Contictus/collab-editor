package rest

import (
	"sync"
	"time"
)

// RateLimiter is an in-memory fixed-window limiter (rate-limit.ts parity).
// Single-node scope, matching the single-process model (invariant #4);
// keyed by IP and email for auth throttles.

type window struct {
	count   int
	resetAt time.Time
}

// RateLimiter guards keys against bursts. Zero value is usable (but shares no
// state until created with NewRateLimiter — prefer the constructor).
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]window
}

// sweepThreshold mirrors SWEEP_THRESHOLD (prune only when large).
const sweepThreshold = 5000

// NewRateLimiter creates an empty limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{windows: make(map[string]window)}
}

// Allow consumes one unit: false once limit is reached within window.
// retryAfter rounds up the seconds until reset (0 when allowed).
func (l *RateLimiter) Allow(key string, limit int, windowMs int64) (ok bool, retryAfter int64) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.windows) >= sweepThreshold {
		for k, w := range l.windows {
			if !now.Before(w.resetAt) {
				delete(l.windows, k)
			}
		}
	}
	w, found := l.windows[key]
	if !found || !now.Before(w.resetAt) {
		l.windows[key] = window{count: 1, resetAt: now.Add(time.Duration(windowMs) * time.Millisecond)}
		return true, 0
	}
	if w.count >= limit {
		return false, int64((w.resetAt.Sub(now) + time.Second - 1) / time.Second)
	}
	w.count++
	l.windows[key] = w
	return true, 0
}

// Reset clears all windows (tests).
func (l *RateLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.windows = make(map[string]window)
}
