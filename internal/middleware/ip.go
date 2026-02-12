package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ExtractIP returns the client IP address from the request.
// It checks X-Forwarded-For first (taking the first IP if comma-separated),
// then falls back to X-Real-IP, and finally to RemoteAddr.
// Returns the IP without port.
//
// SECURITY WARNING: This function trusts X-Forwarded-For and X-Real-IP headers.
// Only use this behind a properly configured reverse proxy (nginx, Cloudflare, etc.)
// that validates and sets these headers correctly. If the application is exposed
// directly to the internet, attackers can spoof these headers to bypass rate limiting.
//
// Recommended reverse proxy configuration:
// - nginx: use real_ip module with set_real_ip_from for trusted proxies
// - Cloudflare: automatically sets correct X-Forwarded-For
// - AWS ALB/ELB: use X-Forwarded-For with proper security groups
func ExtractIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header (alternative proxy header)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	// RemoteAddr is "IP:port", extract just the IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If no port (shouldn't happen), return as-is
		return r.RemoteAddr
	}
	return ip
}
