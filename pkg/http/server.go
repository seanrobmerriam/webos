package http

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Handler is the interface for HTTP request handlers.
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

// HandlerFunc is a function that implements Handler.
type HandlerFunc func(ResponseWriter, *Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

// ResponseWriter is an interface for writing HTTP responses.
type ResponseWriter interface {
	Header() Header
	Write([]byte) (int, error)
	WriteHeader(int)
}

// responseWriter implements ResponseWriter.
type responseWriter struct {
	header      Header
	statusCode  int
	writer      io.Writer
	wroteHeader bool
}

// newResponseWriter creates a new response writer.
func newResponseWriter(w io.Writer) ResponseWriter {
	return &responseWriter{
		header: make(Header),
		writer: w,
	}
}

// Header returns the response headers.
func (rw *responseWriter) Header() Header {
	return rw.header
}

// Write writes the response body.
func (rw *responseWriter) Write(p []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(200)
	}
	return rw.writer.Write(p)
}

// WriteHeader sends the status code and headers.
func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = statusCode
	rw.wroteHeader = true
}

// Server is an HTTP server.
type Server struct {
	Addr           string
	Handler        Handler
	TLSConfig      *tls.Config
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// ListenAndServeTLS starts the HTTPS server.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	addr := s.Addr
	if addr == "" {
		addr = ":https"
	}
	ln, err := tls.Listen("tcp", addr, s.TLSConfig)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve accepts incoming connections and handles them.
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// handleConn handles a single connection.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		if s.ReadTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		}
		req, err := ReadRequest(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			WriteText(conn, StatusBadRequest, err.Error())
			return
		}
		writer := newResponseWriter(conn)
		s.Handler.ServeHTTP(writer, req)
		connClose := req.Header.Get(HeaderConnection)
		if connClose == ConnectionClose {
			return
		}
		if req.Proto == ProtocolHTTP10 {
			return
		}
	}
}

// ServeMux is an HTTP request multiplexer.
type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	hosts map[string]bool
}

// muxEntry holds a handler and its pattern.
type muxEntry struct {
	h       Handler
	pattern string
}

// NewServeMux creates a new ServeMux.
func NewServeMux() *ServeMux {
	return &ServeMux{
		m:     make(map[string]muxEntry),
		hosts: make(map[string]bool),
	}
}

// Handle registers a handler for the given pattern.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if pattern == "" {
		panic("http: invalid pattern")
	}
	if handler == nil {
		panic("http: nil handler")
	}
	mux.m[pattern] = muxEntry{h: handler, pattern: pattern}
}

// HandleFunc registers a handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, fn func(ResponseWriter, *Request)) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.m[pattern] = muxEntry{h: HandlerFunc(fn), pattern: pattern}
}

// ServeHTTP dispatches the request to the handler whose pattern matches.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	if h, ok := mux.m[r.URL.Path]; ok {
		h.h.ServeHTTP(w, r)
		return
	}
	for pattern, e := range mux.m {
		if strings.HasPrefix(pattern, "/") && strings.HasPrefix(r.URL.Path, pattern) {
			e.h.ServeHTTP(w, r)
			return
		}
	}
	NotFound(w, r)
}

// NotFound writes a 404 Not Found response.
func NotFound(w ResponseWriter, r *Request) {
	WriteText(w, StatusNotFound, "404 page not found")
}

// MethodNotAllowed writes a 405 Method Not Allowed response.
func MethodNotAllowed(w ResponseWriter, r *Request) {
	w.Header().Set(HeaderAllow, "GET, POST, PUT, DELETE, OPTIONS")
	WriteText(w, StatusMethodNotAllowed, "method not allowed")
}

// FileServer returns a handler that serves HTTP requests with file contents.
func FileServer(root string) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		urlPath := r.URL.Path
		if urlPath == "/" {
			urlPath = "/index.html"
		}
		filePath := filepath.Join(root, path.Clean(urlPath))
		if !strings.HasPrefix(filePath, root) {
			NotFound(w, r)
			return
		}
		f, err := os.Open(filePath)
		if err != nil {
			NotFound(w, r)
			return
		}
		defer f.Close()
		ct := contentType(filePath)
		w.Header().Set(HeaderContentType, ct)
		w.WriteHeader(200)
		io.Copy(w, f)
	})
}

// contentType returns the MIME type for the given file.
func contentType(file string) string {
	ext := filepath.Ext(file)
	switch strings.ToLower(ext) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// Redirect writes a redirect response.
func Redirect(w ResponseWriter, r *Request, urlStr string, code int) {
	if code < 300 || code > 399 {
		code = StatusFound
	}
	w.Header().Set(HeaderLocation, urlStr)
	w.WriteHeader(code)
	io.WriteString(w, fmt.Sprintf("<a href=\"%s\">%s</a>\n", urlStr, StatusText(code)))
}

// TimeoutHandler returns a handler that runs h with the given time limit.
func TimeoutHandler(h Handler, dt time.Duration, msg string) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		done := make(chan struct{})
		go func() {
			h.ServeHTTP(w, r)
			close(done)
		}()
		select {
		case <-done:
			return
		case <-time.After(dt):
			w.Header().Set(HeaderConnection, ConnectionClose)
			WriteText(w, StatusServiceUnavailable, msg)
		}
	})
}

// StripPrefix returns a handler that strips the given prefix from the URL path.
func StripPrefix(prefix string, h Handler) Handler {
	if prefix == "" {
		return h
	}
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		if strings.HasPrefix(r.URL.Path, prefix) {
			r.URL.Path = r.URL.Path[len(prefix):]
		}
		h.ServeHTTP(w, r)
	})
}

// StaticFile returns a handler that serves static files.
func StaticFile(dir string) Handler {
	return StripPrefix("/static/", FileServer(dir))
}

// APIHandler creates a handler for API routes.
func APIHandler(routes map[string]map[string]Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		methodRoutes, ok := routes[r.URL.Path]
		if !ok {
			NotFound(w, r)
			return
		}
		handler, ok := methodRoutes[r.Method]
		if !ok {
			MethodNotAllowed(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
