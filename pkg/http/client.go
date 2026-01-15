package http

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RoundTripper is an interface representing the ability to execute a
// single HTTP transaction, obtaining the Response for a given Request.
type RoundTripper interface {
	RoundTrip(*Request) (*Response, error)
}

// DefaultClient is the default Client.
var DefaultClient = &Client{
	Transport: &defaultTransport{},
	Timeout:   DefaultClientTimeout,
}

// Client is an HTTP client.
type Client struct {
	Transport RoundTripper
	Timeout   time.Duration
}

// Do sends an HTTP request and returns an HTTP response.
func (c *Client) Do(req *Request) (*Response, error) {
	transport := c.Transport
	if transport == nil {
		transport = DefaultClient.Transport
	}
	return transport.RoundTrip(req)
}

// Get sends a GET request and returns the response.
func (c *Client) Get(urlStr string) (*Response, error) {
	req, err := NewRequest(MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post sends a POST request with the given body.
func (c *Client) Post(urlStr string, contentType string, body io.Reader) (*Response, error) {
	req, err := NewRequest(MethodPost, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderContentType, contentType)
	return c.Do(req)
}

// Head sends a HEAD request and returns the response.
func (c *Client) Head(urlStr string) (*Response, error) {
	req, err := NewRequest(MethodHead, urlStr, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// defaultTransport implements RoundTripper for plain HTTP.
type defaultTransport struct{}

// RoundTrip executes a single HTTP transaction.
func (t *defaultTransport) RoundTrip(req *Request) (*Response, error) {
	u := req.URL
	host := u.Host
	if host == "" {
		host = req.Host
	}
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	var conn net.Conn
	var err error
	if u.Scheme == "https" {
		conn, err = tls.Dial("tcp", net.JoinHostPort(host, port), nil)
	} else {
		conn, err = net.Dial("tcp", net.JoinHostPort(host, port))
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := req.WriteTo(conn); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	return ReadResponse(reader, req)
}

// PooledTransport is a RoundTripper that reuses connections.
type PooledTransport struct {
	pool      map[string][]net.Conn
	connMutex sync.Mutex
	maxIdle   int
	timeout   time.Duration
}

// NewPooledTransport creates a new PooledTransport.
func NewPooledTransport(maxIdle int, timeout time.Duration) *PooledTransport {
	return &PooledTransport{
		pool:    make(map[string][]net.Conn),
		maxIdle: maxIdle,
		timeout: timeout,
	}
}

// RoundTrip executes a single HTTP transaction with connection pooling.
func (t *PooledTransport) RoundTrip(req *Request) (*Response, error) {
	u := req.URL
	host := u.Host
	if host == "" {
		host = req.Host
	}
	key := u.Scheme + "://" + host
	conn, err := t.getConn(key)
	if err != nil {
		conn, err = t.dial(req)
		if err != nil {
			return nil, err
		}
	}
	if _, err := req.WriteTo(conn); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	resp, err := ReadResponse(reader, req)
	if err != nil {
		return nil, err
	}
	connClose := resp.Header.Get(HeaderConnection)
	if connClose == ConnectionClose || connClose == "" {
		conn.Close()
		return resp, nil
	}
	t.putConn(key, conn)
	return resp, nil
}

func (t *PooledTransport) getConn(key string) (net.Conn, error) {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if list, ok := t.pool[key]; ok && len(list) > 0 {
		conn := list[len(list)-1]
		t.pool[key] = list[:len(list)-1]
		if t.timeout > 0 {
			conn.SetReadDeadline(time.Now().Add(t.timeout))
		}
		return conn, nil
	}
	return nil, nil
}

func (t *PooledTransport) putConn(key string, conn net.Conn) {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	if t.maxIdle <= 0 {
		conn.Close()
		return
	}
	list := t.pool[key]
	if len(list) >= t.maxIdle {
		conn.Close()
		return
	}
	t.pool[key] = append(list, conn)
}

func (t *PooledTransport) dial(req *Request) (net.Conn, error) {
	u := req.URL
	host := u.Host
	if host == "" {
		host = req.Host
	}
	port := u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	if u.Scheme == "https" {
		return tls.Dial("tcp", net.JoinHostPort(host, port), nil)
	}
	return net.Dial("tcp", net.JoinHostPort(host, port))
}

func (t *PooledTransport) CloseIdleConnections() {
	t.connMutex.Lock()
	defer t.connMutex.Unlock()
	for _, list := range t.pool {
		for _, conn := range list {
			conn.Close()
		}
	}
	t.pool = make(map[string][]net.Conn)
}

// URLEncode encodes a map of parameters as URL query string.
func URLEncode(data map[string]string) string {
	var parts []string
	for k, v := range data {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	return strings.Join(parts, "&")
}

// URLDecode decodes a URL query string into a map.
func URLDecode(query string) (map[string]string, error) {
	m := make(map[string]string)
	if query == "" {
		return m, nil
	}
	for _, pair := range strings.Split(query, "&") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			k, err := url.QueryUnescape(parts[0])
			if err != nil {
				return nil, err
			}
			v, err := url.QueryUnescape(parts[1])
			if err != nil {
				return nil, err
			}
			m[k] = v
		}
	}
	return m, nil
}

// GetBytes is a convenience method that makes a GET request.
func GetBytes(urlStr string) ([]byte, error) {
	resp, err := DefaultClient.Get(urlStr)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

// GetString is a convenience method that makes a GET request.
func GetString(urlStr string) (string, error) {
	data, err := GetBytes(urlStr)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PostForm submits a form POST request with URL-encoded data.
func PostForm(urlStr string, data map[string]string) (*Response, error) {
	body := strings.NewReader(URLEncode(data))
	return DefaultClient.Post(urlStr, "application/x-www-form-urlencoded", body)
}

// ReadAll reads all data from the reader.
func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
