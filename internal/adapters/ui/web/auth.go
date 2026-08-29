package web

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// AuthMiddleware returns an HTTP middleware that validates authentication tokens.
// If token is empty, authentication is disabled and all requests pass through.
// Loopback/localhost requests bypass authentication by default.
// Remote requests require either 'Authorization: Bearer <token>' header or '?token=<token>' query param.
func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check loopback bypass
			if isLoopback(r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}

			// Check Authorization header
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				bearer := strings.TrimPrefix(authHeader, "Bearer ")
				if bearer == token {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check query parameter
			if qToken := r.URL.Query().Get("token"); qToken != "" {
				if qToken == token {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Unauthorized response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized: invalid or missing access token",
			})
		})
	}
}

func isLoopback(remoteAddr string) bool {
	if remoteAddr == "" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	// Remove bracket if IPv6
	host = strings.Trim(host, "[]")

	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "fe80::1" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}
