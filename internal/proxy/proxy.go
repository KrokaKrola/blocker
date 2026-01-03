package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/user/blocker/internal/blocker"
)

// Server represents the proxy server
type Server struct {
	httpServer *http.Server
	handler    *Handler
	blocker    *blocker.Blocker
	addr       string
}

// New creates a new proxy server
func New(bind string, port int, b *blocker.Blocker) *Server {
	addr := fmt.Sprintf("%s:%d", bind, port)
	handler := NewHandler(b)

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		handler: handler,
		blocker: b,
		addr:    addr,
	}
}

// Start starts the proxy server
func (s *Server) Start() error {
	log.Printf("[proxy] Starting proxy server on %s", s.addr)
	
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("proxy server error: %w", err)
	}
	return nil
}

// Stop gracefully stops the proxy server
func (s *Server) Stop() error {
	log.Println("[proxy] Stopping proxy server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.addr
}
