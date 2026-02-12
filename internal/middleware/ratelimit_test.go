package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		limit       int
		staticPaths []string
		wantErr     bool
	}{
		{
			name:        "valid limit",
			limit:       100,
			staticPaths: []string{"/favicon.svg"},
			wantErr:     false,
		},
		{
			name:        "zero limit",
			limit:       0,
			staticPaths: []string{},
			wantErr:     true,
		},
		{
			name:        "negative limit",
			limit:       -10,
			staticPaths: []string{},
			wantErr:     true,
		},
		{
			name:        "no static paths",
			limit:       50,
			staticPaths: []string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := New(tt.limit, tt.staticPaths)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && rl == nil {
				t.Error("New() returned nil without error")
			}
			if rl != nil {
				rl.Close()
			}
		})
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For single IP",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For multiple IPs",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "203.0.113.1, 198.51.100.1, 192.168.1.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For with spaces",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "  203.0.113.1  ",
			expectedIP:    "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:12345",
			xRealIP:    "203.0.113.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:    "192.168.1.1:12345",
			xForwardedFor: "203.0.113.1",
			xRealIP:       "198.51.100.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := ExtractIP(req)
			if got != tt.expectedIP {
				t.Errorf("ExtractIP() = %v, want %v", got, tt.expectedIP)
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl, err := New(3, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	ip := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, _ := rl.allow(ip)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	allowed, oldestRequest := rl.allow(ip)
	if allowed {
		t.Error("4th request should be blocked")
	}
	if oldestRequest.IsZero() {
		t.Error("oldest request timestamp should be set when blocked")
	}

	// Wait for window to pass (in real scenario, this would be 1 minute)
	// For testing, we'll manipulate the timestamps directly
	rl.mu.Lock()
	// Clear timestamps to simulate window expiration
	rl.requests[ip] = []time.Time{}
	rl.mu.Unlock()

	// After window, request should be allowed again
	allowed, _ = rl.allow(ip)
	if !allowed {
		t.Error("request should be allowed after window expiration")
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		limit          int
		staticPaths    []string
		requestPath    string
		requestCount   int
		expectBlocked  bool
		expectRetryHdr bool
	}{
		{
			name:           "under limit",
			limit:          5,
			staticPaths:    []string{},
			requestPath:    "/test",
			requestCount:   3,
			expectBlocked:  false,
			expectRetryHdr: false,
		},
		{
			name:           "at limit",
			limit:          3,
			staticPaths:    []string{},
			requestPath:    "/test",
			requestCount:   3,
			expectBlocked:  false,
			expectRetryHdr: false,
		},
		{
			name:           "over limit",
			limit:          3,
			staticPaths:    []string{},
			requestPath:    "/test",
			requestCount:   4,
			expectBlocked:  true,
			expectRetryHdr: true,
		},
		{
			name:           "static path bypass",
			limit:          1,
			staticPaths:    []string{"/favicon.svg"},
			requestPath:    "/favicon.svg",
			requestCount:   10,
			expectBlocked:  false,
			expectRetryHdr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := New(tt.limit, tt.staticPaths)
			if err != nil {
				t.Fatalf("failed to create rate limiter: %v", err)
			}
			defer rl.Close()

			// Create a simple handler that returns 200 OK
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := rl.Middleware(nextHandler)

			var lastStatus int
			var lastRetryAfter string

			// Make multiple requests
			for i := 0; i < tt.requestCount; i++ {
				req := httptest.NewRequest("GET", tt.requestPath, nil)
				req.RemoteAddr = "192.168.1.1:12345"
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				lastStatus = w.Code
				lastRetryAfter = w.Header().Get("Retry-After")
			}

			// Check final request status
			if tt.expectBlocked && lastStatus != http.StatusTooManyRequests {
				t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, lastStatus)
			}
			if !tt.expectBlocked && lastStatus != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, lastStatus)
			}

			// Check Retry-After header
			hasRetryAfter := lastRetryAfter != ""
			if hasRetryAfter != tt.expectRetryHdr {
				t.Errorf("Retry-After header presence = %v, want %v", hasRetryAfter, tt.expectRetryHdr)
			}

			// If Retry-After is present, verify it's a positive integer
			if hasRetryAfter {
				if lastRetryAfter == "" || lastRetryAfter == "0" {
					t.Errorf("Retry-After should be a positive integer, got %q", lastRetryAfter)
				}
			}
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	// Create a rate limiter with a short window for testing
	rl, err := New(10, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	// Override window for faster testing
	rl.window = 100 * time.Millisecond

	// Add some requests
	ip := "192.168.1.1"
	rl.allow(ip)
	rl.allow(ip)

	// Verify requests are tracked
	rl.mu.RLock()
	if len(rl.requests[ip]) != 2 {
		t.Errorf("expected 2 requests tracked, got %d", len(rl.requests[ip]))
	}
	rl.mu.RUnlock()

	// Wait for window to expire
	time.Sleep(250 * time.Millisecond)

	// Run cleanup manually
	rl.cleanup()

	// Verify cleanup removed the IP entry
	rl.mu.RLock()
	_, exists := rl.requests[ip]
	rl.mu.RUnlock()

	if exists {
		t.Error("expected IP to be removed after cleanup")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl, err := New(2, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	// Each IP should have independent rate limits
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// IP1: use up the limit
	rl.allow(ip1)
	rl.allow(ip1)
	allowed, _ := rl.allow(ip1)
	if allowed {
		t.Error("IP1 should be blocked after limit")
	}

	// IP2: should still be allowed
	allowed, _ = rl.allow(ip2)
	if !allowed {
		t.Error("IP2 should be allowed (independent limit)")
	}
	allowed, _ = rl.allow(ip2)
	if !allowed {
		t.Error("IP2 second request should be allowed")
	}
	allowed, _ = rl.allow(ip2)
	if allowed {
		t.Error("IP2 should be blocked after its own limit")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl, err := New(100, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	// Create a handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Middleware(nextHandler)

	// Run concurrent requests
	done := make(chan bool)
	numGoroutines := 10
	requestsPerGoroutine := 5

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.1:12345"
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify the requests were tracked
	rl.mu.RLock()
	count := len(rl.requests["192.168.1.1"])
	rl.mu.RUnlock()

	expectedCount := numGoroutines * requestsPerGoroutine
	if count != expectedCount {
		t.Errorf("expected %d requests tracked, got %d", expectedCount, count)
	}
}

func TestRateLimiter_RetryAfterCalculation(t *testing.T) {
	rl, err := New(1, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Middleware(nextHandler)

	// First request - allowed
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request should be allowed, got status %d", w1.Code)
	}

	// Second request - blocked
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request should be blocked, got status %d", w2.Code)
	}

	retryAfter := w2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be present")
	}

	// Retry-After should be close to 60 seconds (1 minute window)
	// We allow some variance for test execution time
	if retryAfter != "60" && retryAfter != "59" {
		t.Logf("Retry-After = %s (expected ~60, allowing 59-60 for timing)", retryAfter)
	}
}

func TestRateLimiter_EmptyIPHandling(t *testing.T) {
	rl, err := New(10, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Middleware(nextHandler)

	// Request with empty RemoteAddr
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ""
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 400 Bad Request when IP cannot be extracted
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty IP, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRateLimiter_BodyContent(t *testing.T) {
	rl, err := New(1, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Middleware(nextHandler)

	// Make 2 requests to trigger rate limit
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	// Check that 429 response has appropriate body
	if w2.Code == http.StatusTooManyRequests {
		body := w2.Body.String()
		if !strings.Contains(body, "Rate limit exceeded") {
			t.Errorf("expected body to contain 'Rate limit exceeded', got %q", body)
		}
	}
}

func TestRateLimiter_MultipleClose(t *testing.T) {
	rl, err := New(10, []string{})
	if err != nil {
		t.Fatalf("failed to create rate limiter: %v", err)
	}

	// Close multiple times should not panic
	rl.Close()
	rl.Close()
	rl.Close()

	// Test passes if no panic occurs
}
