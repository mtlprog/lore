package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheControl(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedHeader string
	}{
		// Static images - 1 year immutable
		{
			name:           "favicon.svg",
			method:         "GET",
			path:           "/favicon.svg",
			expectedHeader: "public, max-age=31536000, immutable",
		},
		{
			name:           "og-image.png",
			method:         "GET",
			path:           "/og-image.png",
			expectedHeader: "public, max-age=31536000, immutable",
		},
		{
			name:           "apple-touch-icon",
			method:         "GET",
			path:           "/apple-touch-icon.png",
			expectedHeader: "public, max-age=31536000, immutable",
		},
		{
			name:           "skill.md",
			method:         "GET",
			path:           "/skill.md",
			expectedHeader: "public, max-age=31536000, immutable",
		},

		// robots.txt - 1 day
		{
			name:           "robots.txt",
			method:         "GET",
			path:           "/robots.txt",
			expectedHeader: "public, max-age=86400",
		},

		// Swagger - 1 hour
		{
			name:           "swagger docs",
			method:         "GET",
			path:           "/swagger/doc.json",
			expectedHeader: "public, max-age=3600",
		},
		{
			name:           "swagger ui",
			method:         "GET",
			path:           "/swagger/index.html",
			expectedHeader: "public, max-age=3600",
		},

		// Init forms (GET) - 1 hour
		{
			name:           "init landing",
			method:         "GET",
			path:           "/init",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "init participant form",
			method:         "GET",
			path:           "/init/participant",
			expectedHeader: "public, max-age=3600",
		},
		{
			name:           "init corporate form",
			method:         "GET",
			path:           "/init/corporate",
			expectedHeader: "public, max-age=3600",
		},

		// API endpoints - 1 minute
		{
			name:           "api stats",
			method:         "GET",
			path:           "/api/v1/stats",
			expectedHeader: "public, max-age=60, must-revalidate",
		},
		{
			name:           "api accounts",
			method:         "GET",
			path:           "/api/v1/accounts",
			expectedHeader: "public, max-age=60, must-revalidate",
		},
		{
			name:           "api account detail",
			method:         "GET",
			path:           "/api/v1/accounts/GABC123",
			expectedHeader: "public, max-age=60, must-revalidate",
		},
		{
			name:           "api reputation",
			method:         "GET",
			path:           "/api/v1/accounts/GABC123/reputation",
			expectedHeader: "public, max-age=60, must-revalidate",
		},
		{
			name:           "api relationships",
			method:         "GET",
			path:           "/api/v1/accounts/GABC123/relationships",
			expectedHeader: "public, max-age=60, must-revalidate",
		},
		{
			name:           "api search",
			method:         "GET",
			path:           "/api/v1/search",
			expectedHeader: "public, max-age=60, must-revalidate",
		},

		// HTML pages - 5 minutes
		{
			name:           "home page",
			method:         "GET",
			path:           "/",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "account detail",
			method:         "GET",
			path:           "/accounts/GABC123",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "account reputation",
			method:         "GET",
			path:           "/accounts/GABC123/reputation",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "transaction",
			method:         "GET",
			path:           "/transactions/abc123",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "search",
			method:         "GET",
			path:           "/search",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "token detail",
			method:         "GET",
			path:           "/tokens/GABC123/MTLAP",
			expectedHeader: "public, max-age=300, must-revalidate",
		},

		// POST requests - no cache
		{
			name:           "post init participant",
			method:         "POST",
			path:           "/init/participant",
			expectedHeader: "no-store",
		},
		{
			name:           "post init corporate",
			method:         "POST",
			path:           "/init/corporate",
			expectedHeader: "no-store",
		},

		// HEAD requests - same as GET
		{
			name:           "head home",
			method:         "HEAD",
			path:           "/",
			expectedHeader: "public, max-age=300, must-revalidate",
		},
		{
			name:           "head static image",
			method:         "HEAD",
			path:           "/favicon.svg",
			expectedHeader: "public, max-age=31536000, immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple handler that returns 200 OK
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := CacheControl(nextHandler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			gotHeader := w.Header().Get("Cache-Control")
			if gotHeader != tt.expectedHeader {
				t.Errorf("Cache-Control = %q, want %q", gotHeader, tt.expectedHeader)
			}
		})
	}
}

func TestCacheControl_NonGETMethods(t *testing.T) {
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := CacheControl(nextHandler)

			req := httptest.NewRequest(method, "/api/v1/accounts", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			gotHeader := w.Header().Get("Cache-Control")
			expectedHeader := "no-store"
			if gotHeader != expectedHeader {
				t.Errorf("%s request: Cache-Control = %q, want %q", method, gotHeader, expectedHeader)
			}
		})
	}
}

func TestIsStaticImage(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "favicon.svg",
			path:     "/favicon.svg",
			expected: true,
		},
		{
			name:     "og-image.svg",
			path:     "/og-image.svg",
			expected: true,
		},
		{
			name:     "og-image.png",
			path:     "/og-image.png",
			expected: true,
		},
		{
			name:     "favicon-32x32.png",
			path:     "/favicon-32x32.png",
			expected: true,
		},
		{
			name:     "favicon-16x16.png",
			path:     "/favicon-16x16.png",
			expected: true,
		},
		{
			name:     "apple-touch-icon.png",
			path:     "/apple-touch-icon.png",
			expected: true,
		},
		{
			name:     "skill.md",
			path:     "/skill.md",
			expected: true,
		},
		{
			name:     "non-static file",
			path:     "/accounts/GABC123",
			expected: false,
		},
		{
			name:     "api endpoint",
			path:     "/api/v1/stats",
			expected: false,
		},
		{
			name:     "home page",
			path:     "/",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStaticImage(tt.path)
			if got != tt.expected {
				t.Errorf("isStaticImage(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
