package http

import (
	"bufio"
	"bytes"
	"net/url"
	"strings"
	"testing"
)

// TestHeaderOperations tests Header operations.
func TestHeaderOperations(t *testing.T) {
	h := make(Header)
	h.Set("Content-Type", "text/plain")
	h.Add("Accept", "text/html")
	h.Add("Accept", "application/json")

	if h.Get("Content-Type") != "text/plain" {
		t.Errorf("expected text/plain, got %s", h.Get("Content-Type"))
	}
	if h.Get("content-type") != "text/plain" {
		t.Errorf("case-insensitive lookup failed")
	}
	if len(h["Accept"]) != 2 {
		t.Errorf("expected 2 Accept values, got %d", len(h["Accept"]))
	}
	h.Del("Content-Type")
	if h.Get("Content-Type") != "" {
		t.Errorf("expected empty after delete")
	}
}

// TestCanonicalHeaderKey tests header key canonicalization.
func TestCanonicalHeaderKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"content-type", "Content-Type"},
		{"accept-encoding", "Accept-Encoding"},
		{"x-custom-header", "X-Custom-Header"},
		{"contentlength", "Contentlength"},
		{"host", "Host"},
	}
	for _, tt := range tests {
		result := CanonicalHeaderKey(tt.input)
		if result != tt.expected {
			t.Errorf("CanonicalHeaderKey(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestParseRequestLine tests request line parsing.
func TestParseRequestLine(t *testing.T) {
	tests := []struct {
		line    string
		method  string
		path    string
		proto   string
		wantErr bool
	}{
		{"GET / HTTP/1.1", "GET", "/", "HTTP/1.1", false},
		{"POST /api/data HTTP/1.1", "POST", "/api/data", "HTTP/1.1", false},
		{"GET / HTTP/1.0", "GET", "/", "HTTP/1.0", false},
		{"INVALID", "", "", "", true},
		{"GET /", "", "", "", true},
	}
	for _, tt := range tests {
		method, path, proto, err := ParseRequestLine(tt.line)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseRequestLine(%q) expected error, got nil", tt.line)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRequestLine(%q) unexpected error: %v", tt.line, err)
			continue
		}
		if method != tt.method || path != tt.path || proto != tt.proto {
			t.Errorf("ParseRequestLine(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.line, method, path, proto, tt.method, tt.path, tt.proto)
		}
	}
}

// TestParseStatusLine tests status line parsing.
func TestParseStatusLine(t *testing.T) {
	tests := []struct {
		line    string
		proto   string
		status  int
		message string
		wantErr bool
	}{
		{"HTTP/1.1 200 OK", "HTTP/1.1", 200, "OK", false},
		{"HTTP/1.1 404 Not Found", "HTTP/1.1", 404, "Not Found", false},
		{"HTTP/1.1 500", "HTTP/1.1", 500, "", false},
		{"HTTP/1.1", "", 0, "", true},
		{"200 OK", "", 0, "", true},
	}
	for _, tt := range tests {
		proto, status, message, err := ParseStatusLine(tt.line)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseStatusLine(%q) expected error, got nil", tt.line)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseStatusLine(%q) unexpected error: %v", tt.line, err)
			continue
		}
		if proto != tt.proto || status != tt.status || message != tt.message {
			t.Errorf("ParseStatusLine(%q) = (%q, %d, %q), want (%q, %d, %q)",
				tt.line, proto, status, message, tt.proto, tt.status, tt.message)
		}
	}
}

// TestReadHeaders tests header reading.
func TestReadHeaders(t *testing.T) {
	input := "Content-Type: text/html\r\nContent-Length: 123\r\n\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	headers, err := ReadHeaders(reader)
	if err != nil {
		t.Fatalf("ReadHeaders error: %v", err)
	}
	if headers.Get("Content-Type") != "text/html" {
		t.Errorf("Content-Type = %q, want %q", headers.Get("Content-Type"), "text/html")
	}
	if headers.Get("Content-Length") != "123" {
		t.Errorf("Content-Length = %q, want %q", headers.Get("Content-Length"), "123")
	}
}

// TestURLEncoding tests URL encoding and decoding.
func TestURLEncoding(t *testing.T) {
	data := map[string]string{
		"name":  "John Doe",
		"query": "hello world",
	}
	encoded := URLEncode(data)
	if !strings.Contains(encoded, "name=John+Doe") {
		t.Errorf("URLEncode = %q, expected name=John+Doe", encoded)
	}
	decoded, err := URLDecode(encoded)
	if err != nil {
		t.Fatalf("URLDecode error: %v", err)
	}
	if decoded["name"] != "John Doe" {
		t.Errorf("decoded name = %q, want %q", decoded["name"], "John Doe")
	}
}

// TestIsValidMethod tests method validation.
func TestIsValidMethod(t *testing.T) {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"}
	for _, m := range validMethods {
		if !isValidMethod(m) {
			t.Errorf("isValidMethod(%q) = false, want true", m)
		}
	}
	invalidMethods := []string{"", "GET ", "GET\r", "INVALID METHOD"}
	for _, m := range invalidMethods {
		if isValidMethod(m) {
			t.Errorf("isValidMethod(%q) = true, want false", m)
		}
	}
}

// TestStatusText tests status text generation.
func TestStatusText(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "OK"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{301, "Moved Permanently"},
		{999, "Unknown"},
	}
	for _, tt := range tests {
		result := StatusText(tt.code)
		if result != tt.expected {
			t.Errorf("StatusText(%d) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

// TestHeaderClone tests header cloning.
func TestHeaderClone(t *testing.T) {
	original := make(Header)
	original.Set("Content-Type", "text/html")
	original.Add("Set-Cookie", "a=1")
	original.Add("Set-Cookie", "b=2")

	clone := original.Clone()
	clone.Set("Content-Type", "text/plain")
	clone.Add("Set-Cookie", "c=3")

	if original.Get("Content-Type") != "text/html" {
		t.Errorf("original Content-Type changed")
	}
	if len(original["Set-Cookie"]) != 2 {
		t.Errorf("original Set-Cookie changed")
	}
	if clone.Get("Content-Type") != "text/plain" {
		t.Errorf("clone Content-Type not changed")
	}
	if len(clone["Set-Cookie"]) != 3 {
		t.Errorf("clone Set-Cookie count = %d, want 3", len(clone["Set-Cookie"]))
	}
}

// TestParseQuery tests query string parsing.
func TestParseQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected url.Values
	}{
		{"", url.Values{}},
		{"a=1", url.Values{"a": []string{"1"}}},
		{"a=1&b=2", url.Values{"a": []string{"1"}, "b": []string{"2"}}},
		{"a=1&a=2", url.Values{"a": []string{"1", "2"}}},
	}
	for _, tt := range tests {
		result := parseQuery(tt.query)
		for k, v := range tt.expected {
			if len(result[k]) != len(v) {
				t.Errorf("parseQuery(%q)[%q] = %v, want %v", tt.query, k, result[k], v)
			}
		}
	}
}

// TestWriteHeaders tests header writing.
func TestWriteHeaders(t *testing.T) {
	h := make(Header)
	h.Set("Content-Type", "text/html")
	h.Set("Content-Length", "123")

	var buf bytes.Buffer
	if _, err := h.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Content-Type: text/html\r\n") {
		t.Errorf("WriteTo output missing Content-Type header")
	}
	if !strings.Contains(output, "Content-Length: 123\r\n") {
		t.Errorf("WriteTo output missing Content-Length header")
	}
}

// BenchmarkHeaderGet benchmarks Header.Get operation.
func BenchmarkHeaderGet(b *testing.B) {
	h := make(Header)
	h.Set("Content-Type", "text/html")
	h.Add("Accept", "text/html")
	h.Add("Accept", "application/json")
	h.Add("Accept", "text/plain")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Get("Content-Type")
		_ = h.Get("accept")
	}
}

// BenchmarkCanonicalHeaderKey benchmarks header key canonicalization.
func BenchmarkCanonicalHeaderKey(b *testing.B) {
	headers := []string{
		"content-type",
		"accept-encoding",
		"x-custom-header",
		"authorization",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, h := range headers {
			_ = CanonicalHeaderKey(h)
		}
	}
}
