package httpapi

import (
	"strings"
	"sync"
	"time"
)

// RateLimiter is the fixed-window limiter the workshop login route uses to cap
// failed attempts per client (AC-29). The login ticket wires it into handleLogin.
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

// NewRateLimiter builds a limiter allowing at most limit events per window per key.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records one attempt for key and reports whether it is within the limit.
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.limit {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	return true
}

// AllowLogin records one sign-in attempt for the client's IP and, when present,
// its e-mail, and reports whether both are still within the limit (AC-29). When
// the IP is already over the limit the e-mail slot is left untouched.
func (rl *RateLimiter) AllowLogin(ip, email string) bool {
	ip = strings.TrimSpace(ip)
	if ip != "" && !rl.Allow("ip:"+ip) {
		return false
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" && !rl.Allow("email:"+email) {
		return false
	}
	return true
}
