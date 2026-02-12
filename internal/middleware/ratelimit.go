package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/samber/lo"
)

const (
	windowDuration  = 1 * time.Minute
	cleanupInterval = 1 * time.Minute
)

// RateLimiter wraps an http.Handler with rate limiting per IP address.
type RateLimiter struct {
	limit       int                    // Maximum requests per window
	window      time.Duration          // Time window for rate limiting
	requests    map[string][]time.Time // IP -> request timestamps
	mu          sync.RWMutex           // Thread-safe access
	cleanupDone chan struct{}          // Shutdown signal for cleanup goroutine
	closeOnce   sync.Once              // Ensures Close() is called only once
	staticPaths map[string]bool        // Paths that bypass rate limiting
}

// New creates a new rate limiter with the given limit and static paths to bypass.
// Returns error if limit is invalid.
//
// IMPORTANT: Close() must be called when shutting down to stop the background
// cleanup goroutine and prevent goroutine leaks.
func New(limit int, staticPaths []string) (*RateLimiter, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("rate limit must be positive, got %d", limit)
	}

	// Build static paths map for O(1) lookup
	staticMap := make(map[string]bool, len(staticPaths))
	for _, path := range staticPaths {
		staticMap[path] = true
	}

	rl := &RateLimiter{
		limit:       limit,
		window:      windowDuration,
		requests:    make(map[string][]time.Time),
		cleanupDone: make(chan struct{}),
		staticPaths: staticMap,
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	slog.Info("rate limiter initialized",
		"limit", limit,
		"window", windowDuration.String(),
		"static_paths", len(staticPaths),
	)

	return rl, nil
}

// Middleware returns an http.Handler that wraps the next handler with rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for static files
		if rl.staticPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Extract client IP
		ip := ExtractIP(r)
		if ip == "" {
			// Shouldn't happen, but handle gracefully
			slog.Warn("failed to extract IP from request", "path", r.URL.Path)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Check rate limit
		allowed, oldestRequest := rl.allow(ip)
		if !allowed {
			// Calculate Retry-After in seconds
			now := time.Now()
			retryAfter := int(rl.window.Seconds() - now.Sub(oldestRequest).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			slog.Debug("rate limit exceeded",
				"ip", ip,
				"path", r.URL.Path,
				"limit", rl.limit,
			)

			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Request allowed, proceed
		next.ServeHTTP(w, r)
	})
}

// allow checks if a request from the given IP is allowed.
// Returns (allowed, oldestRequestTime).
// If not allowed, oldestRequestTime is the timestamp of the oldest request in the window.
func (rl *RateLimiter) allow(ip string) (bool, time.Time) {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get existing requests for this IP
	timestamps := rl.requests[ip]

	// Filter to only keep requests within the window (sliding window)
	validTimestamps := filterValidTimestamps(timestamps, cutoff)

	// Check if limit exceeded
	if len(validTimestamps) >= rl.limit {
		// Return oldest timestamp for Retry-After calculation
		return false, validTimestamps[0]
	}

	// Add current request timestamp
	validTimestamps = append(validTimestamps, now)
	rl.requests[ip] = validTimestamps

	return true, time.Time{}
}

// cleanupLoop runs in the background and periodically removes stale entries.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.cleanupDone:
			return
		}
	}
}

// cleanup removes expired entries from the requests map.
// Removes IPs with no requests in the last window to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	now := time.Now()
	// Remove entries older than 1x window (matches the sliding window used in allow)
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, timestamps := range rl.requests {
		// Filter valid timestamps
		validTimestamps := filterValidTimestamps(timestamps, cutoff)

		// Remove IP if no valid requests remain
		if len(validTimestamps) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validTimestamps
		}
	}
}

// filterValidTimestamps filters timestamps to only keep those after the cutoff.
// Uses lo.Filter for readable, idiomatic code.
func filterValidTimestamps(timestamps []time.Time, cutoff time.Time) []time.Time {
	return lo.Filter(timestamps, func(ts time.Time, _ int) bool {
		return ts.After(cutoff)
	})
}

// Close stops the background cleanup goroutine.
// MUST be called when shutting down the server to prevent goroutine leaks.
// Safe to call multiple times.
func (rl *RateLimiter) Close() {
	rl.closeOnce.Do(func() {
		close(rl.cleanupDone)
	})
}
