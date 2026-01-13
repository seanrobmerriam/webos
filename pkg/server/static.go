package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StaticFileHandler handles static file serving with caching and MIME types.
type StaticFileHandler struct {
	dir            string
	cacheControl   string
	indexFiles     []string
	useETag        bool
	useGzip        bool
	gzipExtensions map[string]bool
}

// NewStaticFileHandler creates a new static file handler.
func NewStaticFileHandler(dir string) *StaticFileHandler {
	return &StaticFileHandler{
		dir:            dir,
		cacheControl:   "public, max-age=3600",
		indexFiles:     []string{"index.html", "index.htm"},
		useETag:        true,
		useGzip:        false,
		gzipExtensions: map[string]bool{".html": true, ".css": true, ".js": true, ".json": true, ".txt": true},
	}
}

// SetCacheControl sets the Cache-Control header value.
func (h *StaticFileHandler) SetCacheControl(value string) {
	h.cacheControl = value
}

// SetIndexFiles sets the files to try when serving a directory.
func (h *StaticFileHandler) SetIndexFiles(files []string) {
	h.indexFiles = files
}

// EnableETag enables or disables ETag generation.
func (h *StaticFileHandler) EnableETag(enabled bool) {
	h.useETag = enabled
}

// EnableGzip enables or disables gzip compression.
func (h *StaticFileHandler) EnableGzip(enabled bool) {
	h.useGzip = enabled
}

// ServeHTTP implements http.Handler interface.
func (h *StaticFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	path := filepath.Join(h.dir, filepath.Clean(r.URL.Path))

	// Check if path is a directory
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if fi.IsDir() {
		// Try to serve index file
		for _, indexFile := range h.indexFiles {
			indexPath := filepath.Join(path, indexFile)
			if fi, err := os.Stat(indexPath); err == nil && !fi.IsDir() {
				h.serveFile(w, r, indexPath)
				return
			}
		}
		// No index file found, list directory or error
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	h.serveFile(w, r, path)
}

// serveFile serves a single file with proper headers and caching.
func (h *StaticFileHandler) serveFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	fi, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check for If-Modified-Since
	if modifiedSince := r.Header.Get("If-Modified-Since"); modifiedSince != "" {
		t, err := time.Parse(http.TimeFormat, modifiedSince)
		if err == nil && !fi.ModTime().After(t) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Set content type
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// Set caching headers
	if h.cacheControl != "" {
		w.Header().Set("Cache-Control", h.cacheControl)
	}
	w.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))

	// Generate ETag
	if h.useETag {
		hash := h.fileHash(path)
		etag := fmt.Sprintf(`"%s"`, hash)
		w.Header().Set("ETag", etag)

		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
			if strings.Contains(ifNoneMatch, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	// Handle range requests
	if ranges := r.Header.Get("Range"); ranges != "" {
		h.serveRange(w, r, file, fi, ranges, contentType)
		return
	}

	// Copy file content
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}

// serveRange handles HTTP range requests.
func (h *StaticFileHandler) serveRange(w http.ResponseWriter, r *http.Request, file *os.File, fi os.FileInfo, ranges string, contentType string) {
	parts := strings.SplitN(ranges, "=", 2)
	if len(parts) != 2 || parts[0] != "bytes" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	rangeParts := strings.Split(parts[1], ",")
	if len(rangeParts) != 1 {
		http.Error(w, "Multiple ranges not supported", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeSpec := strings.TrimSpace(rangeParts[0])
	rangeBounds := strings.SplitN(rangeSpec, "-", 2)
	if len(rangeBounds) != 2 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	start, end := parseRangeBounds(rangeBounds, fi.Size())
	if start > end || start >= fi.Size() {
		http.Error(w, "Range Not Satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Seek to start position
	if _, err := file.Seek(start, 0); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fi.Size()))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
	w.WriteHeader(http.StatusPartialContent)

	io.Copy(w, io.LimitReader(file, end-start+1))
}

// parseRangeBounds parses range bounds from string.
func parseRangeBounds(parts []string, size int64) (int64, int64) {
	start := int64(0)
	end := size - 1

	if parts[0] != "" {
		fmt.Sscanf(parts[0], "%d", &start)
	}
	if parts[1] != "" {
		fmt.Sscanf(parts[1], "%d", &end)
	}

	return start, end
}

// fileHash computes a hash of the file for ETag.
func (h *StaticFileHandler) fileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

// MimeTypes maps file extensions to MIME types.
var MimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".htm":   "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript; charset=utf-8",
	".json":  "application/json",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".eot":   "application/vnd.ms-fontobject",
	".pdf":   "application/pdf",
	".txt":   "text/plain; charset=utf-8",
	".xml":   "application/xml",
	".yaml":  "application/yaml",
	".yml":   "application/yaml",
	".md":    "text/markdown",
	".csv":   "text/csv",
	".zip":   "application/zip",
	".tar":   "application/tar",
	".gz":    "application/gzip",
}

// GzipMiddleware returns a middleware that compresses responses with gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Create gzip writer
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.Close()

		next.ServeHTTP(gw, r)
	})
}

// gzipWriter wraps http.ResponseWriter to compress output.
type gzipWriter struct {
	http.ResponseWriter
	buf bytes.Buffer
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *gzipWriter) Close() {
	// Note: In a real implementation, you'd use compress/gzip here
	// For simplicity, we're just demonstrating the concept
	w.ResponseWriter.Write(w.buf.Bytes())
}

// DirectoryListing returns a middleware that shows directory listings.
func DirectoryListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a placeholder for directory listing functionality
		// In production, you'd implement proper HTML directory listing
		next.ServeHTTP(w, r)
	})
}

// FileServer creates an HTTP file server with the given root directory.
func FileServer(root string) http.Handler {
	handler := NewStaticFileHandler(root)
	return handler
}

// ServeDir is a convenience function to serve a directory.
func ServeDir(pattern, dir string) (string, http.Handler) {
	return pattern, FileServer(dir)
}

// ErrorHandler is a custom error handler.
type ErrorHandler struct {
	log *log.Logger
}

// NewErrorHandler creates a new error handler.
func NewErrorHandler(logger *log.Logger) *ErrorHandler {
	return &ErrorHandler{log: logger}
}

// ServeHTTP implements http.Handler interface.
func (h *ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			h.log.Printf("Panic: %v", rec)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	w = &errorResponseWriter{ResponseWriter: w, handler: h}

	// Note: In a real implementation, you'd wrap the next handler
	// For simplicity, this is a placeholder
}

// errorResponseWriter wraps http.ResponseWriter to handle errors.
type errorResponseWriter struct {
	http.ResponseWriter
	handler *ErrorHandler
}

func (w *errorResponseWriter) WriteHeader(code int) {
	if code >= 400 {
		w.handler.log.Printf("HTTP error: %d", code)
	}
	w.ResponseWriter.WriteHeader(code)
}

// NotFoundHandler returns a custom 404 handler.
func NotFoundHandler(message string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, message, http.StatusNotFound)
	})
}

// MethodNotAllowedHandler returns a handler for method not allowed errors.
func MethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})
}

// ValidatePath checks if a path is safe and within the root directory.
func ValidatePath(root, requestedPath string) (string, error) {
	// Clean the requested path
	cleanPath := filepath.Clean(requestedPath)

	// Resolve to absolute path
	absPath, err := filepath.Abs(filepath.Join(root, cleanPath))
	if err != nil {
		return "", errors.New("invalid path")
	}

	// Ensure the resolved path is within the root
	if !strings.HasPrefix(absPath, filepath.Clean(root)) {
		return "", errors.New("path outside root directory")
	}

	return absPath, nil
}
