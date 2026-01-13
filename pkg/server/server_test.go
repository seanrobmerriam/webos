package server

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServer_New(t *testing.T) {
	cfg := Config{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	srv := New(cfg)
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.Addr() != ":8080" {
		t.Errorf("expected addr ':8080', got '%s'", srv.Addr())
	}
}

func TestServer_NewWithAddr(t *testing.T) {
	srv := NewWithAddr(":9090")
	if srv == nil {
		t.Fatal("NewWithAddr() returned nil")
	}
	if srv.Addr() != ":9090" {
		t.Errorf("expected addr ':9090', got '%s'", srv.Addr())
	}
}

func TestServer_SetHandler(t *testing.T) {
	srv := New(Config{Addr: ":8080"})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})

	srv.SetHandler(handler)
}

func TestServer_SetStaticDir(t *testing.T) {
	srv := New(Config{Addr: ":8080"})
	srv.SetStaticDir("./static")

	srv.mu.RLock()
	if srv.staticDir != "./static" {
		t.Errorf("expected static dir './static', got '%s'", srv.staticDir)
	}
	if srv.staticHandler == nil {
		t.Error("expected staticHandler to be set")
	}
	srv.mu.RUnlock()
}

func TestServer_ServeHTTP(t *testing.T) {
	srv := New(Config{Addr: ":8080"})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	srv.SetHandler(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestServer_StaticFile(t *testing.T) {
	// Create a temporary directory with a test file
	tmpDir, err := os.MkdirTemp("", "static-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.html")
	if err := os.WriteFile(testFile, []byte("<html>Test</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	srv := New(Config{Addr: ":8080"})
	srv.SetStaticDir(tmpDir)

	req := httptest.NewRequest(http.MethodGet, "/test.html", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("expected content-type 'text/html', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestServer_NotFound(t *testing.T) {
	srv := New(Config{Addr: ":8080"})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestServer_HealthHandler(t *testing.T) {
	handler := HealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestServer_ReadyHandler(t *testing.T) {
	handler := ReadyHandler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestServer_MetricsHandler(t *testing.T) {
	handler := MetricsHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected content-type 'text/plain', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestServer_Shutdown(t *testing.T) {
	srv := New(Config{Addr: ":8080"})

	// Start the server in a goroutine
	go func() {
		srv.ListenAndServe()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)

	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func TestServer_Close(t *testing.T) {
	srv := New(Config{Addr: ":8080"})

	// Start the server in a goroutine
	go func() {
		srv.ListenAndServe()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Close the server
	err := srv.Close()

	if err != nil {
		t.Errorf("close error: %v", err)
	}
}

func TestServer_Started(t *testing.T) {
	srv := New(Config{Addr: ":8080"})

	if srv.Started() {
		t.Error("expected server not to be started")
	}
}

func TestStaticFileHandler_New(t *testing.T) {
	handler := NewStaticFileHandler("./static")
	if handler == nil {
		t.Fatal("NewStaticFileHandler() returned nil")
	}
	if handler.dir != "./static" {
		t.Errorf("expected dir './static', got '%s'", handler.dir)
	}
}

func TestStaticFileHandler_SetCacheControl(t *testing.T) {
	handler := NewStaticFileHandler("./static")
	handler.SetCacheControl("public, max-age=7200")
	if handler.cacheControl != "public, max-age=7200" {
		t.Errorf("expected cache control 'public, max-age=7200', got '%s'", handler.cacheControl)
	}
}

func TestStaticFileHandler_EnableETag(t *testing.T) {
	handler := NewStaticFileHandler("./static")
	handler.EnableETag(false)
	if handler.useETag {
		t.Error("expected ETag to be disabled")
	}
}

func TestStaticFileHandler_EnableGzip(t *testing.T) {
	handler := NewStaticFileHandler("./static")
	handler.EnableGzip(true)
	if !handler.useGzip {
		t.Error("expected gzip to be enabled")
	}
}

func TestStaticFileHandler_ServeHTTP_NotFound(t *testing.T) {
	handler := NewStaticFileHandler("./nonexistent")

	req := httptest.NewRequest(http.MethodGet, "/test.html", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestStaticFileHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "static-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	handler := NewStaticFileHandler(tmpDir)

	req := httptest.NewRequest(http.MethodPost, "/test.html", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestFileServer(t *testing.T) {
	handler := FileServer("./static")
	if handler == nil {
		t.Fatal("FileServer() returned nil")
	}
}

func TestServeDir(t *testing.T) {
	pattern, handler := ServeDir("/static", "./static")
	if pattern != "/static" {
		t.Errorf("expected pattern '/static', got '%s'", pattern)
	}
	if handler == nil {
		t.Error("expected handler to be non-nil")
	}
}

func TestNotFoundHandler(t *testing.T) {
	handler := NotFoundHandler("Custom not found")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestMethodNotAllowedHandler(t *testing.T) {
	handler := MethodNotAllowedHandler()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestValidatePath(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test valid path
	path, err := ValidatePath(tmpDir, "sub/file.txt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !filepath.HasPrefix(path, tmpDir) {
		t.Errorf("path should be within root")
	}

	// Test path traversal attempt
	_, err = ValidatePath(tmpDir, "../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal attempt")
	}
}

func TestTLSConfig_LoadCertificates(t *testing.T) {
	cfg := TLSConfig{
		CertFile: "server.crt",
		KeyFile:  "server.key",
	}

	_, err := cfg.LoadCertificates()
	// This will fail because the files don't exist, but that's expected
	if err == nil {
		t.Error("expected error for missing certificate files")
	}
}

func TestServerTLSConfig(t *testing.T) {
	certs := []tls.Certificate{}
	config := ServerTLSConfig(certs)

	if config == nil {
		t.Fatal("ServerTLSConfig() returned nil")
	}
	if config.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3, got version %d", config.MinVersion)
	}
}

func TestClientTLSConfig(t *testing.T) {
	config, err := ClientTLSConfig("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("ClientTLSConfig() returned nil")
	}
}

func TestClientTLSConfig_WithCA(t *testing.T) {
	// This test expects a valid CA certificate, which we don't have
	// Skip this test as it requires a valid CA certificate
	t.Skip("Requires a valid CA certificate file")
}

func TestWithMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	middlewareCalled := false
	middleware := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			h.ServeHTTP(w, r)
		})
	}

	wrapped := WithMiddleware(handler, middleware)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("middleware was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequestLogger(t *testing.T) {
	logger := log.New(os.Stderr, "test: ", 0)
	middleware := RequestLogger(logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	middleware := TimeoutMiddleware(100 * time.Millisecond)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("OK"))
	})

	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(100)
	middleware := limiter.Middleware()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDirectoryListing(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	handler := DirectoryListing(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGzipMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	handler := GzipMiddleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDefaultServer(t *testing.T) {
	srv := DefaultServer()
	if srv == nil {
		t.Fatal("DefaultServer() returned nil")
	}
	if srv.Addr() != ":8080" {
		t.Errorf("expected addr ':8080', got '%s'", srv.Addr())
	}
}

func TestErrorHandler(t *testing.T) {
	logger := log.New(os.Stderr, "test: ", 0)
	handler := NewErrorHandler(logger)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestMimeTypes(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".html", "text/html; charset=utf-8"},
		{".css", "text/css; charset=utf-8"},
		{".js", "application/javascript; charset=utf-8"},
		{".json", "application/json"},
		{".png", "image/png"},
	}

	for _, tt := range tests {
		if MimeTypes[tt.ext] != tt.expected {
			t.Errorf("expected MIME type '%s' for '%s', got '%s'", tt.expected, tt.ext, MimeTypes[tt.ext])
		}
	}
}
