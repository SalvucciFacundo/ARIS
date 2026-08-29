package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"aris/internal/adapters/ui/web/static"
	"aris/internal/core/services"
)

// Config configures the web server adapter.
type Config struct {
	Host     string
	Port     int
	Token    string
	AutoPort bool
	StaticFS http.FileSystem
}

// Server encapsulates the ARIS HTTP web server.
type Server struct {
	cfg      Config
	agent    *services.AgentService
	broker   *SSEBroker
	handlers *Handlers
	httpSrv  *http.Server
	listener net.Listener
	mu       sync.Mutex
	actualAddr string
	actualPort int
}

// NewServer constructs a new web Server instance.
func NewServer(cfg Config, agent *services.AgentService) *Server {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.StaticFS == nil {
		cfg.StaticFS = static.FS()
	}
	broker := NewSSEBroker()
	handlers := NewHandlers(agent, broker, cfg)
	return &Server{
		cfg:      cfg,
		agent:    agent,
		broker:   broker,
		handlers: handlers,
	}
}

// Broker returns the SSE broker instance.
func (s *Server) Broker() *SSEBroker {
	return s.broker
}

// Handlers returns the HTTP handlers instance.
func (s *Server) Handlers() *Handlers {
	return s.handlers
}

// Addr returns the bound host:port address string.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actualAddr
}

// Port returns the bound listening port.
func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actualPort
}

// URL returns the full HTTP listening URL.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actualAddr == "" {
		return ""
	}
	return fmt.Sprintf("http://%s", s.actualAddr)
}

// Start binds to the designated address and serves incoming HTTP requests.
func (s *Server) Start(ctx context.Context) error {
	router := NewRouter(s.cfg, s.handlers, s.broker)

	var ln net.Listener
	var err error

	host := s.cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}

	startPort := s.cfg.Port
	maxAttempts := 1
	if s.cfg.AutoPort && startPort > 0 {
		maxAttempts = 10
	}

	for i := 0; i < maxAttempts; i++ {
		tryPort := startPort
		if startPort > 0 {
			tryPort = startPort + i
		}
		bindAddr := fmt.Sprintf("%s:%d", host, tryPort)
		ln, err = net.Listen("tcp", bindAddr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("bind web server to %s:%d (attempts %d): %w", host, startPort, maxAttempts, err)
	}

	s.mu.Lock()
	s.listener = ln
	s.actualAddr = ln.Addr().String()
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
		s.actualPort = tcpAddr.Port
	}
	s.httpSrv = &http.Server{
		Handler:      router,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	s.mu.Unlock()

	// Start SSE broker heartbeat in background
	go s.broker.Start(ctx)

	// Watch for context cancellation
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdownCtx)
	}()

	err = s.httpSrv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.broker != nil {
		s.broker.closeAll()
	}

	if s.httpSrv == nil {
		return nil
	}

	return s.httpSrv.Shutdown(ctx)
}
