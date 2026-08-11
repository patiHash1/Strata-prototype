package utils

import (
	"net/http"
	"sync"
	"time"
)

// rateLimiter is a simple in-memory per-IP token bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // requests per window
	window   time.Duration // time window
	cleanup  time.Duration // how often to purge stale entries
}

type visitor struct {
	tokens   int
	lastSeen time.Time
	resetAt  time.Time
}

// newRateLimiter creates a rate limiter that allows `rate` requests per `window`
// per unique client IP. Stale entries are cleaned up periodically.
func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
		cleanup:  window * 2,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.cleanup {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// allow returns true if the request from the given IP should be allowed.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.After(v.resetAt) {
		rl.visitors[ip] = &visitor{
			tokens:   rl.rate - 1,
			lastSeen: now,
			resetAt:  now.Add(rl.window),
		}
		return true
	}

	v.lastSeen = now
	if v.tokens <= 0 {
		return false
	}
	v.tokens--
	return true
}

// RateLimitMiddleware returns middleware that limits requests per client IP.
// It uses the X-Forwarded-For header (first value) if present, otherwise
// falls back to RemoteAddr.
func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	rl := newRateLimiter(requestsPerMinute, time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !rl.allow(ip) {
				WriteErr(w, http.StatusTooManyRequests, "rate limit exceeded, try again later")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from the request, preferring X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (the original client).
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	// Strip port from RemoteAddr (e.g. "192.168.1.1:12345" → "192.168.1.1").
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
