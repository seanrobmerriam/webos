package http

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
)

// Request represents an HTTP request.
type Request struct {
	Method  string
	URL     *url.URL
	Proto   string
	Header  Header
	Body    io.Reader
	Host    string
	Trailer Header
}

// NewRequest creates a new HTTP request.
func NewRequest(method, urlStr string, body io.Reader) (*Request, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	return &Request{
		Method: method,
		URL:    u,
		Proto:  ProtocolHTTP11,
		Header: make(Header),
		Body:   body,
	}, nil
}

// WriteTo writes the request to the given writer in HTTP format.
func (r *Request) WriteTo(w io.Writer) (int64, error) {
	var n int64
	// Write request line
	cnt, err := io.WriteString(w, r.Method+" "+r.RequestURI()+" "+r.Proto+"\r\n")
	n += int64(cnt)
	if err != nil {
		return n, err
	}
	// Write headers
	if r.Host != "" {
		cnt, err := io.WriteString(w, HeaderHost+": "+r.Host+"\r\n")
		n += int64(cnt)
		if err != nil {
			return n, err
		}
	}
	// Write headers using Header.WriteTo
	headerCnt, headerErr := r.Header.WriteTo(w)
	n += headerCnt
	if headerErr != nil {
		return n, headerErr
	}
	// Write end of headers
	cnt, err = io.WriteString(w, "\r\n")
	n += int64(cnt)
	if err != nil {
		return n, err
	}
	// Write body
	if r.Body != nil {
		cnt, err := io.Copy(w, r.Body)
		n += cnt
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// RequestURI returns the request URI (path and query).
func (r *Request) RequestURI() string {
	if r.URL != nil {
		return r.URL.RequestURI()
	}
	return "/"
}

// Clone returns a deep copy of the request.
func (r *Request) Clone() *Request {
	clone := &Request{
		Method:  r.Method,
		URL:     r.URL,
		Proto:   r.Proto,
		Header:  r.Header.Clone(),
		Host:    r.Host,
		Trailer: r.Trailer.Clone(),
	}
	if r.Body != nil {
		clone.Body = ioutil.NopCloser(bytes.NewReader(mustReadAll(r.Body)))
	}
	return clone
}

// mustReadAll reads all data from the reader.
func mustReadAll(r io.Reader) []byte {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil
	}
	return data
}

// ParseRequestLine parses an HTTP request line.
// Returns method, path, and protocol.
func ParseRequestLine(line string) (string, string, string, error) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return "", "", "", &ProtocolError{"malformed request line: " + line}
	}
	return parts[0], parts[1], parts[2], nil
}

// ReadRequest reads an HTTP request from the reader.
func ReadRequest(r *bufio.Reader) (*Request, error) {
	// Read request line
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if line == "" {
		return nil, io.EOF
	}
	method, path, proto, err := ParseRequestLine(line)
	if err != nil {
		return nil, err
	}
	if !isValidMethod(method) {
		return nil, &ProtocolError{"invalid method: " + method}
	}
	// Parse URL
	u, err := url.Parse(path)
	if err != nil {
		return nil, &ProtocolError{"malformed URL: " + path}
	}
	// Read headers
	headers, err := ReadHeaders(r)
	if err != nil {
		return nil, err
	}
	// Get host
	host := headers.Get(HeaderHost)
	if host == "" {
		host = u.Host
	}
	// Check for chunked transfer encoding
	te := headers.Get(HeaderTransferEncoding)
	var body io.Reader = r
	if te == TransferEncodingChunked {
		body = &chunkedReader{r: r}
	}
	req := &Request{
		Method: method,
		URL:    u,
		Proto:  proto,
		Header: headers,
		Body:   body,
		Host:   host,
	}
	return req, nil
}

// TransferEncoding returns the Transfer-Encoding header value.
func (r *Request) TransferEncoding() string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(HeaderTransferEncoding)
}

// ContentLength returns the Content-Length header value, or -1 if not set.
func (r *Request) ContentLength() int64 {
	if r.Header == nil {
		return -1
	}
	cl := r.Header.Get(HeaderContentLength)
	if cl == "" {
		return -1
	}
	var n int64
	for _, c := range []byte(cl) {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

// UserAgent returns the User-Agent header value.
func (r *Request) UserAgent() string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(HeaderUserAgent)
}

// Referer returns the Referer header value.
func (r *Request) Referer() string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(HeaderReferer)
}

// cookies returns all cookies from the Cookie header.
func (r *Request) cookies() map[string]string {
	cookies := make(map[string]string)
	if r.Header == nil {
		return cookies
	}
	cookieHeader := r.Header.Get(HeaderCookie)
	for _, pair := range strings.Split(cookieHeader, ";") {
		pair = strings.TrimSpace(pair)
		idx := strings.IndexByte(pair, '=')
		if idx > 0 {
			name := strings.TrimSpace(pair[:idx])
			value := strings.TrimSpace(pair[idx+1:])
			cookies[name] = value
		}
	}
	return cookies
}

// Cookie returns the value of the named cookie.
func (r *Request) Cookie(name string) string {
	return r.cookies()[name]
}

// chunkedReader implements io.Reader for chunked transfer encoding.
type chunkedReader struct {
	r    *bufio.Reader
	err  error
	left int64 // bytes remaining in current chunk
}

func (cr *chunkedReader) Read(p []byte) (n int, err error) {
	if cr.err != nil {
		return 0, cr.err
	}
	for cr.left == 0 {
		// Read chunk size line
		line, err := cr.r.ReadString('\n')
		if err != nil {
			cr.err = err
			return 0, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		// Find semicolon for extensions
		if idx := strings.IndexByte(line, ';'); idx >= 0 {
			line = line[:idx]
		}
		// Parse chunk size
		size, err := parseChunkSize(line)
		if err != nil {
			cr.err = err
			return 0, err
		}
		cr.left = int64(size)
		if cr.left == 0 {
			// Last chunk
			// Read trailing headers
			_, err := ReadHeaders(cr.r)
			if err != nil {
				cr.err = err
			}
			return 0, cr.err
		}
	}
	// Read chunk data
	if int64(len(p)) > cr.left {
		p = p[:cr.left]
	}
	n, err = io.ReadFull(cr.r, p)
	cr.left -= int64(n)
	if cr.left == 0 {
		// Read CRLF after chunk
		_, err := cr.r.ReadString('\n')
		if err != nil && cr.err == nil {
			cr.err = err
		}
	}
	return n, cr.err
}

func parseChunkSize(s string) (int, error) {
	size := 0
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
			size = size*16 + int(c-'0')
		case c >= 'a' && c <= 'f':
			size = size*16 + int(c-'a'+10)
		case c >= 'A' && c <= 'F':
			size = size*16 + int(c-'A'+10)
		default:
			return 0, &ProtocolError{"invalid chunk size: " + s}
		}
	}
	return size, nil
}
