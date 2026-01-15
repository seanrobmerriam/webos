package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
)

// Response represents an HTTP response.
type Response struct {
	Status     string
	StatusCode int
	Proto      string
	Header     Header
	Body       io.Reader
	Trailer    Header
	Request    *Request
}

// NewResponse creates a new HTTP response.
func NewResponse(statusCode int, body io.Reader) *Response {
	return &Response{
		Status:     fmt.Sprintf("%d %s", statusCode, StatusText(statusCode)),
		StatusCode: statusCode,
		Proto:      ProtocolHTTP11,
		Header:     make(Header),
		Body:       body,
	}
}

// WriteTo writes the response to the given writer in HTTP format.
func (r *Response) WriteTo(w io.Writer) (int64, error) {
	var n int64
	cnt, err := io.WriteString(w, r.Proto+" "+r.Status+"\r\n")
	n += int64(cnt)
	if err != nil {
		return n, err
	}
	// Write headers using Header.WriteTo
	headerCnt, headerErr := r.Header.WriteTo(w)
	n += headerCnt
	if headerErr != nil {
		return n, headerErr
	}
	cnt, err = io.WriteString(w, "\r\n")
	n += int64(cnt)
	if err != nil {
		return n, err
	}
	if r.Body != nil {
		cnt, err := io.Copy(w, r.Body)
		n += cnt
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Clone returns a deep copy of the response.
func (r *Response) Clone() *Response {
	clone := &Response{
		Status:     r.Status,
		StatusCode: r.StatusCode,
		Proto:      r.Proto,
		Header:     r.Header.Clone(),
		Trailer:    r.Trailer.Clone(),
		Request:    r.Request,
	}
	if r.Body != nil {
		clone.Body = ioutil.NopCloser(bytes.NewReader(mustReadAll(r.Body)))
	}
	return clone
}

// StatusText returns the standard text for the given status code.
func StatusText(code int) string {
	switch code {
	case StatusContinue:
		return "Continue"
	case StatusSwitchingProtocols:
		return "Switching Protocols"
	case StatusOK:
		return "OK"
	case StatusCreated:
		return "Created"
	case StatusAccepted:
		return "Accepted"
	case StatusNoContent:
		return "No Content"
	case StatusMovedPermanently:
		return "Moved Permanently"
	case StatusFound:
		return "Found"
	case StatusSeeOther:
		return "See Other"
	case StatusNotModified:
		return "Not Modified"
	case StatusBadRequest:
		return "Bad Request"
	case StatusUnauthorized:
		return "Unauthorized"
	case StatusForbidden:
		return "Forbidden"
	case StatusNotFound:
		return "Not Found"
	case StatusMethodNotAllowed:
		return "Method Not Allowed"
	case StatusInternalServerError:
		return "Internal Server Error"
	case StatusNotImplemented:
		return "Not Implemented"
	case StatusBadGateway:
		return "Bad Gateway"
	case StatusServiceUnavailable:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}

// ReadResponse reads an HTTP response from the reader.
func ReadResponse(r *bufio.Reader, request *Request) (*Response, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if line == "" {
		return nil, io.EOF
	}
	proto, statusCode, message, err := ParseStatusLine(line)
	if err != nil {
		return nil, err
	}
	headers, err := ReadHeaders(r)
	if err != nil {
		return nil, err
	}
	te := headers.Get(HeaderTransferEncoding)
	var body io.Reader = r
	if te == TransferEncodingChunked {
		body = &chunkedReader{r: r}
	}
	status := strconv.Itoa(statusCode)
	if message != "" {
		status = fmt.Sprintf("%d %s", statusCode, message)
	}
	resp := &Response{
		Status:     status,
		StatusCode: statusCode,
		Proto:      proto,
		Header:     headers,
		Body:       body,
		Request:    request,
	}
	return resp, nil
}

// ParseStatusLine parses an HTTP status line.
func ParseStatusLine(line string) (string, int, string, error) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return "", 0, "", &ProtocolError{"malformed status line: " + line}
	}
	proto := parts[0]
	statusCode, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, "", &ProtocolError{"invalid status code: " + parts[1]}
	}
	message := ""
	if len(parts) > 2 {
		message = parts[2]
	}
	return proto, statusCode, message, nil
}

// ContentLength returns the Content-Length header value.
func (r *Response) ContentLength() int64 {
	if r.Header == nil {
		return -1
	}
	cl := r.Header.Get(HeaderContentLength)
	if cl == "" {
		return -1
	}
	n, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return -1
	}
	return n
}

// TransferEncoding returns the Transfer-Encoding header value.
func (r *Response) TransferEncoding() string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(HeaderTransferEncoding)
}

// Location returns the Location header value.
func (r *Response) Location() string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(HeaderLocation)
}

// SetCookie adds a Set-Cookie header.
func (r *Response) SetCookie(name, value string) {
	r.Header.Add(HeaderSetCookie, name+"="+value)
}

// WriteText writes a plain text response.
func WriteText(w io.Writer, statusCode int, text string) error {
	header := make(Header)
	header.Set(HeaderContentType, "text/plain; charset=utf-8")
	header.Set(HeaderContentLength, strconv.Itoa(len(text)))
	return WriteResponse(w, statusCode, header, strings.NewReader(text))
}

// WriteHTML writes an HTML response.
func WriteHTML(w io.Writer, statusCode int, html string) error {
	header := make(Header)
	header.Set(HeaderContentType, "text/html; charset=utf-8")
	header.Set(HeaderContentLength, strconv.Itoa(len(html)))
	return WriteResponse(w, statusCode, header, strings.NewReader(html))
}

// WriteJSON writes a JSON response.
func WriteJSON(w io.Writer, statusCode int, json []byte) error {
	header := make(Header)
	header.Set(HeaderContentType, "application/json")
	header.Set(HeaderContentLength, strconv.Itoa(len(json)))
	return WriteResponse(w, statusCode, header, bytes.NewReader(json))
}

// WriteRedirect writes a redirect response.
func WriteRedirect(w io.Writer, statusCode int, location string) error {
	header := make(Header)
	header.Set(HeaderLocation, location)
	return WriteResponse(w, statusCode, header, nil)
}

// WriteResponse writes a complete response with headers and body.
func WriteResponse(w io.Writer, statusCode int, header Header, body io.Reader) error {
	if err := WriteStatus(w, ProtocolHTTP11, statusCode, ""); err != nil {
		return err
	}
	if header != nil {
		if _, err := header.WriteTo(w); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}
	if body != nil {
		if _, err := io.Copy(w, body); err != nil {
			return err
		}
	}
	return nil
}

// WriteStatus writes the status line.
func WriteStatus(w io.Writer, proto string, statusCode int, message string) error {
	status := strconv.Itoa(statusCode)
	if message != "" {
		status = fmt.Sprintf("%d %s", statusCode, message)
	}
	_, err := io.WriteString(w, proto+" "+status+"\r\n")
	return err
}
