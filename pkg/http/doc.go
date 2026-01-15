/*
Package http provides HTTP/1.1 and HTTP/2 client and server implementations.

This package implements a complete HTTP stack from scratch using only the Go
standard library. It includes:

  - HTTP/1.1 client and server with keep-alive support
  - HTTP/2 support with multiplexing and HPACK header compression
  - TLS integration for HTTPS support

# HTTP/1.1 Features

  - Request/response parsing and generation
  - Chunked transfer encoding
  - Keep-alive connections
  - Pipeline support
  - Request body streaming

# HTTP/2 Features

  - Binary framing layer
  - Multiplexed streams
  - HPACK header compression
  - Server push support
  - Flow control

# Usage

Client example:

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, _ := client.Do(req)
	defer resp.Body.Close()

Server example:

	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello, World!"))
		}),
	}
	server.ListenAndServe()

For HTTP/2, use TLS with ALPN to negotiate the protocol:

	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{NextProtos: []string{"h2", "http/1.1"}},
		Handler: http.HandlerFunc(handler),
	}
	server.ListenAndServeTLS("cert.pem", "key.pem")
*/
package http
