// Package server provides a production-ready HTTP server with
// static file serving, TLS support, and graceful shutdown.
//
// The server supports:
//   - HTTP/1.1 and HTTP/2 via TLS ALPN
//   - Static file serving with MIME types and caching
//   - Graceful shutdown with context cancellation
//   - Request logging and metrics
//   - TLS 1.3 with strong cipher suites
//
// Example usage:
//
//	srv := server.New(":8080")
//	srv.StaticDir("/", "./static")
//	go srv.ListenAndServe()
//	srv.Shutdown()
package server
