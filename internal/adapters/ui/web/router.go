package web

import (
	"net/http"
	"strings"
)

// NewRouter constructs and configures the HTTP router with routes and auth middleware.
func NewRouter(cfg Config, handlers *Handlers, broker *SSEBroker) http.Handler {
	mux := http.NewServeMux()

	// Assets handler
	var staticHandler http.Handler
	if cfg.StaticFS != nil {
		staticHandler = http.FileServer(cfg.StaticFS)
	} else {
		staticHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("/* placeholder */"))
		})
	}

	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.StripPrefix("/assets/", staticHandler).ServeHTTP(w, r)
	})

	// Core web routes
	mux.HandleFunc("/", handlers.HandleIndex)
	mux.HandleFunc("/api/events", broker.ServeHTTP)
	mux.HandleFunc("/api/generate", handlers.HandleGenerate)
	mux.HandleFunc("/api/inpaint", handlers.HandleInpaint)
	mux.HandleFunc("/api/history", handlers.HandleHistory)
	mux.HandleFunc("/api/image/", handlers.HandleImage)
	mux.HandleFunc("/api/subagents", handlers.HandleSubagents)
	mux.HandleFunc("/api/backends", handlers.HandleBackends)

	// Wrap entire mux with auth middleware (auth middleware itself allows /assets/ and loopback)
	authMw := AuthMiddleware(cfg.Token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			// Skip auth check for public static assets
			mux.ServeHTTP(w, r)
			return
		}
		authMw(mux).ServeHTTP(w, r)
	})
}
