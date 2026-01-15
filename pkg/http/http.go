package http

import (
	"bufio"
	"bytes"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Method constants for HTTP requests.
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
	MethodPatch   = "PATCH"
)

// Common HTTP status codes.
const (
	StatusContinue            = 100
	StatusSwitchingProtocols  = 101
	StatusOK                  = 200
	StatusCreated             = 201
	StatusAccepted            = 202
	StatusNoContent           = 204
	StatusMovedPermanently    = 301
	StatusFound               = 302
	StatusSeeOther            = 303
	StatusNotModified         = 304
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusInternalServerError = 500
	StatusNotImplemented      = 501
	StatusBadGateway          = 502
	StatusServiceUnavailable  = 503
)

// Protocol versions.
const (
	ProtocolHTTP10 = "HTTP/1.0"
	ProtocolHTTP11 = "HTTP/1.1"
	ProtocolHTTP2  = "HTTP/2"
)

// Default timeout values.
const (
	DefaultClientTimeout     = 30 * time.Second
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultWriteTimeout      = 0 // No timeout by default
	DefaultIdleTimeout       = 90 * time.Second
)

// Header names (canonicalized).
const (
	HeaderAccept             = "Accept"
	HeaderAcceptEncoding     = "Accept-Encoding"
	HeaderAllow              = "Allow"
	HeaderAuthorization      = "Authorization"
	HeaderCacheControl       = "Cache-Control"
	HeaderConnection         = "Connection"
	HeaderContentEncoding    = "Content-Encoding"
	HeaderContentLength      = "Content-Length"
	HeaderContentType        = "Content-Type"
	HeaderCookie             = "Cookie"
	HeaderDate               = "Date"
	HeaderHost               = "Host"
	HeaderIfModifiedSince    = "If-Modified-Since"
	HeaderIfNoneMatch        = "If-None-Match"
	HeaderKeepAlive          = "Keep-Alive"
	HeaderLocation           = "Location"
	HeaderProxyAuthenticate  = "Proxy-Authenticate"
	HeaderProxyAuthorization = "Proxy-Authorization"
	HeaderRange              = "Range"
	HeaderReferer            = "Referer"
	HeaderServer             = "Server"
	HeaderSetCookie          = "Set-Cookie"
	HeaderTransferEncoding   = "Transfer-Encoding"
	HeaderUpgrade            = "Upgrade"
	HeaderUserAgent          = "User-Agent"
	HeaderWWWAuthenticate    = "WWW-Authenticate"
	HeaderXForwardedFor      = "X-Forwarded-For"
	HeaderXRealIP            = "X-Real-IP"
)

// Transfer encoding constants.
const (
	TransferEncodingChunked = "chunked"
	TransferEncodingGzip    = "gzip"
	TransferEncodingDeflate = "deflate"
	TransferEncodingBr      = "br"
)

// Connection options.
const (
	ConnectionKeepAlive = "keep-alive"
	ConnectionClose     = "close"
	ConnectionUpgrade   = "Upgrade"
)

// Header represents HTTP headers as a case-insensitive key-value map.
type Header map[string][]string

// Get returns the first value for the given key, case-insensitive.
// Returns empty string if key not found.
func (h Header) Get(key string) string {
	if h == nil {
		return ""
	}
	canonical := CanonicalHeaderKey(key)
	if values, ok := h[canonical]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

// Set sets the header value, replacing any existing values.
func (h Header) Set(key, value string) {
	if h == nil {
		return
	}
	canonical := CanonicalHeaderKey(key)
	h[canonical] = []string{value}
}

// Add adds a new header value to the key.
func (h Header) Add(key, value string) {
	if h == nil {
		return
	}
	canonical := CanonicalHeaderKey(key)
	h[canonical] = append(h[canonical], value)
}

// Del removes all values for the given key.
func (h Header) Del(key string) {
	if h == nil {
		return
	}
	canonical := CanonicalHeaderKey(key)
	delete(h, canonical)
}

// Clone returns a deep copy of the header.
func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	clone := make(Header)
	for k, vv := range h {
		newVV := make([]string, len(vv))
		copy(newVV, vv)
		clone[k] = newVV
	}
	return clone
}

// WriteHeaders writes all headers to the given writer in HTTP format.
func WriteHeaders(w io.Writer, h Header) (int64, error) {
	var n int64
	for k, vv := range h {
		for _, v := range vv {
			line := k + ": " + v + "\r\n"
			cnt, err := io.WriteString(w, line)
			n += int64(cnt)
			if err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

// WriteTo writes the headers to the given writer in HTTP format.
func (h Header) WriteTo(w io.Writer) (int64, error) {
	return WriteHeaders(w, h)
}

// CanonicalHeaderKey returns the canonical format of the header key.
// The first character and any character following a hyphen are uppercased;
// the rest are lowercased. Examples: "content-type" -> "Content-Type".
func CanonicalHeaderKey(s string) string {
	if s == "" {
		return s
	}
	result := make([]byte, len(s))
	upperNext := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if upperNext && c >= 'a' && c <= 'z' {
			result[i] = c - 'a' + 'A'
		} else {
			result[i] = c
		}
		upperNext = (c == '-')
	}
	return string(result)
}

// ReadHeader reads a single header line from the reader.
func ReadHeader(r *bufio.Reader) (string, string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if line == "" {
		return "", "", nil
	}
	idx := strings.IndexByte(line, ':')
	if idx == -1 {
		return "", "", &ProtocolError{"malformed header: " + line}
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	return key, value, nil
}

// ReadHeaders reads all headers from the reader.
func ReadHeaders(r *bufio.Reader) (Header, error) {
	headers := make(Header)
	for {
		key, value, err := ReadHeader(r)
		if err != nil {
			return nil, err
		}
		if key == "" {
			break // End of headers
		}
		headers[key] = append(headers[key], value)
	}
	return headers, nil
}

// ProtocolError represents an HTTP protocol error.
type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

// bufferPool provides a pool of bytes.Buffer objects.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// getBuffer gets a buffer from the pool.
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer returns a buffer to the pool.
func putBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}

// parseQuery parses URL query parameters.
func parseQuery(query string) url.Values {
	values := make(url.Values)
	if query == "" {
		return values
	}
	for _, pair := range strings.Split(query, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key, _ := url.QueryUnescape(parts[0])
			value, _ := url.QueryUnescape(parts[1])
			values.Add(key, value)
		} else if len(parts) == 1 {
			key, _ := url.QueryUnescape(parts[0])
			values.Add(key, "")
		}
	}
	return values
}

// isTokenChar returns true if the byte is a valid token character.
func isTokenChar(c byte) bool {
	return c < 0x80 && tokenChars[c]
}

// tokenChars is a lookup table for valid HTTP token characters.
var tokenChars = [256]bool{
	'!': true, '#': true, '$': true, '%': true, '&': true,
	'\'': true, '*': true, '+': true, '-': true, '.': true,
	'^': true, '_': true, '`': true, '|': true, '~': true,
	'0': true, '1': true, '2': true, '3': true, '4': true,
	'5': true, '6': true, '7': true, '8': true, '9': true,
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true,
	'F': true, 'G': true, 'H': true, 'I': true, 'J': true,
	'K': true, 'L': true, 'M': true, 'N': true, 'O': true,
	'P': true, 'Q': true, 'R': true, 'S': true, 'T': true,
	'U': true, 'V': true, 'W': true, 'X': true, 'Y': true,
	'Z': true, 'a': true, 'b': true, 'c': true, 'd': true,
	'e': true, 'f': true, 'g': true, 'h': true, 'i': true,
	'j': true, 'k': true, 'l': true, 'm': true, 'n': true,
	'o': true, 'p': true, 'q': true, 'r': true, 's': true,
	't': true, 'u': true, 'v': true, 'w': true, 'x': true,
	'y': true, 'z': true,
}

// isValidMethod checks if the method is a valid HTTP method.
func isValidMethod(method string) bool {
	if method == "" {
		return false
	}
	for _, c := range []byte(method) {
		if !isTokenChar(c) {
			return false
		}
	}
	return true
}

// isConnectMethod returns true if the method is CONNECT.
func isConnectMethod(method string) bool {
	return method == MethodConnect
}
