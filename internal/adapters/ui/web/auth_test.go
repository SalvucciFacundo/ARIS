package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	tests := []struct {
		name           string
		token          string
		remoteAddr     string
		authHeader     string
		queryParam     string
		expectedStatus int
	}{
		{
			name:           "no token configured -> allows all",
			token:          "",
			remoteAddr:     "192.168.1.50:1234",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "localhost IPv4 loopback bypasses token",
			token:          "secret-token",
			remoteAddr:     "127.0.0.1:45678",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "localhost IPv6 loopback bypasses token",
			token:          "secret-token",
			remoteAddr:     "[::1]:45678",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "remote client with valid Bearer token",
			token:          "secret-token",
			remoteAddr:     "10.0.0.5:1234",
			authHeader:     "Bearer secret-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "remote client with valid query token",
			token:          "secret-token",
			remoteAddr:     "10.0.0.5:1234",
			queryParam:     "?token=secret-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "remote client with invalid Bearer token",
			token:          "secret-token",
			remoteAddr:     "10.0.0.5:1234",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "remote client with missing token",
			token:          "secret-token",
			remoteAddr:     "10.0.0.5:1234",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := AuthMiddleware(tt.token)
			handler := mw(dummyHandler)

			url := "/api/history" + tt.queryParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
