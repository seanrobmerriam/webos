package server

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Server represents an HTTP server with routing and middleware support.
type Server struct {
	addr          string
	handler       http.Handler
	server        *http.Server
	tlsConfig     *tls.Config
	tlsEnabled    bool
	staticDir     string
	staticHandler http.Handler
	mu            sync.RWMutex
	started       bool
	stopCh        chan struct{}
}

// Config holds server configuration.
type Config struct {
	Addr         string
	Handler      http.Handler
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	TLSConfig    *tls.Config
	StaticDir    string
}

// New creates a new HTTP server with the given configuration.
func New(cfg Config) *Server {
	s := &Server{
		addr:      cfg.Addr,
		handler:   cfg.Handler,
		tlsConfig: cfg.TLSConfig,
		stopCh:    make(chan struct{}),
	}

	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 30 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 30 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 120 * time.Second
	}

	s.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      s,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	if cfg.StaticDir != "" {
		s.staticDir = cfg.StaticDir
		s.staticHandler = NewStaticFileHandler(cfg.StaticDir)
	}

	return s
}

// NewWithAddr creates a new HTTP server listening on the given address.
func NewWithAddr(addr string) *Server {
	return New(Config{Addr: addr})
}

// SetHandler sets the main request handler.
func (s *Server) SetHandler(handler http.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

// SetStaticDir sets the directory for static file serving.
func (s *Server) SetStaticDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.staticDir = dir
	s.staticHandler = NewStaticFileHandler(dir)
}

// EnableTLS enables TLS with the given certificates.
func (s *Server) EnableTLS(certFile, keyFile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	s.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"h2", "http/1.1"},
	}

	s.tlsEnabled = true
	s.server.TLSConfig = s.tlsConfig
	return nil
}

// ServeHTTP implements http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	staticHandler := s.staticHandler
	handler := s.handler
	s.mu.RUnlock()

	// API routes and other requests go to the main handler first
	if handler != nil {
		handler.ServeHTTP(w, r)
		return
	}

	// If no main handler, try static files
	if staticHandler != nil {
		staticHandler.ServeHTTP(w, r)
		return
	}

	// Default handler - return 404
	http.NotFound(w, r)
}

// ListenAndServe starts the server and listens for connections.
func (s *Server) ListenAndServe() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mu.Unlock()

	if s.tlsEnabled {
		return s.server.ListenAndServeTLS("", "")
	}
	return s.server.ListenAndServe()
}

// ListenAndServeTLS starts the server with TLS enabled.
func (s *Server) ListenAndServeTLS() error {
	if !s.tlsEnabled {
		return fmt.Errorf("TLS not enabled")
	}
	return s.server.ListenAndServeTLS("", "")
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	close(s.stopCh)
	return s.server.Shutdown(ctx)
}

// Close closes the server immediately.
func (s *Server) Close() error {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	return s.server.Close()
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.server.Addr
}

// Started returns whether the server has been started.
func (s *Server) Started() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// Serve starts the server and handles shutdown signals.
func (s *Server) Serve() error {
	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.Shutdown(ctx)
	}()

	log.Printf("Starting server on %s", s.Addr())
	return s.ListenAndServe()
}

// HealthHandler returns a handler for health checks.
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})
}

// ReadyHandler returns a handler for readiness checks.
func ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready"}`))
	})
}

// MetricsHandler returns a handler for metrics (placeholder).
func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# No metrics configured\n"))
	})
}

// Command line flags
var (
	listenAddr = flag.String("addr", ":8080", "Server listen address")
	tlsEnabled = flag.Bool("tls", false, "Enable TLS")
	certFile   = flag.String("cert", "server.crt", "TLS certificate file")
	keyFile    = flag.String("key", "server.key", "TLS key file")
	staticDir  = flag.String("static", "./static", "Static files directory")
)

// RunMain runs the server with command line arguments.
func RunMain() error {
	flag.Parse()

	srv := New(Config{
		Addr:      *listenAddr,
		StaticDir: *staticDir,
	})

	if *tlsEnabled {
		if err := srv.EnableTLS(*certFile, *keyFile); err != nil {
			return fmt.Errorf("failed to enable TLS: %w", err)
		}
	}

	// Add health check endpoint
	router := http.NewServeMux()
	router.Handle("/health", HealthHandler())
	router.Handle("/ready", ReadyHandler())
	srv.SetHandler(router)

	return srv.Serve()
}

// DefaultServer returns a server with default configuration.
func DefaultServer() *Server {
	return New(Config{
		Addr:         ":8080",
		Handler:      nil,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})
}

// WithMiddleware wraps a handler with middleware.
func WithMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// RequestLogger returns a middleware that logs requests.
func RequestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

// TimeoutMiddleware returns a middleware that adds a timeout to requests.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}

// RateLimiter is a placeholder for rate limiting.
type RateLimiter struct {
	requestsPerSecond int
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(rps int) *RateLimiter {
	return &RateLimiter{requestsPerSecond: rps}
}

// Middleware returns a rate limiting middleware.
func (r *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Placeholder: In production, implement proper rate limiting
			next.ServeHTTP(w, req)
		})
	}
}
