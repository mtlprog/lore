package middleware

import (
	"net/http"
	"slices"
	"strings"
)

// CacheControl is a middleware that sets Cache-Control headers based on request path.
// Different content types have different caching strategies:
// - Static images: 1 year (immutable)
// - robots.txt: 1 day
// - Swagger docs: 1 hour
// - HTML pages: 5 minutes with revalidation
// - Init forms: 1 hour
// - API endpoints: 1 minute with revalidation
// - POST/PUT/DELETE: no caching
func CacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip cache headers for non-GET requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
			return
		}

		path := r.URL.Path

		// Static images - cache for 1 year (immutable content)
		if isStaticImage(path) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			next.ServeHTTP(w, r)
			return
		}

		// robots.txt - cache for 1 day
		if path == "/robots.txt" {
			w.Header().Set("Cache-Control", "public, max-age=86400")
			next.ServeHTTP(w, r)
			return
		}

		// Swagger docs - cache for 1 hour
		if strings.HasPrefix(path, "/swagger/") {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			next.ServeHTTP(w, r)
			return
		}

		// Init forms (GET only) - cache for 1 hour (static content)
		if strings.HasPrefix(path, "/init/") {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			next.ServeHTTP(w, r)
			return
		}

		// API endpoints - cache for 1 minute with revalidation
		if strings.HasPrefix(path, "/api/") {
			w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
			next.ServeHTTP(w, r)
			return
		}

		// HTML pages - cache for 5 minutes with revalidation
		// Includes: /, /accounts/{id}, /transactions/{hash}, /search, /tokens/{issuer}/{code}
		w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

// isStaticImage checks if the path is a static image file.
func isStaticImage(path string) bool {
	staticImages := []string{
		"/favicon.svg",
		"/og-image.svg",
		"/og-image.png",
		"/favicon-32x32.png",
		"/favicon-16x16.png",
		"/apple-touch-icon.png",
		"/skill.md",
	}

	return slices.Contains(staticImages, path)
}
