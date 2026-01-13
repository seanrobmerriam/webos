# PHASE 3.3: HTTP Client & Server

**Phase Context**: Phase 3 implements the networking stack. This sub-phase creates enhanced HTTP implementation.

**Sub-Phase Objective**: Implement HTTP/1.1 client and server, HTTP/2 with multiplexing, and TLS integration.

**Prerequisites**: 
- Phase 3.1 (Network Stack) recommended
- Phase 3.2 (DNS) recommended

**Integration Point**: HTTP will be used for web services and API endpoints.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a complete HTTP/1.1 and HTTP/2 implementation from scratch.

---

### Directory Structure

```
webos/
└── pkg/
    └── http/
        ├── doc.go              # Package documentation
        ├── http.go             # HTTP types
        ├── request.go          # Request parsing
        ├── response.go         # Response generation
        ├── client.go           # HTTP client
        ├── server.go           # HTTP server
        ├── h2/                 # HTTP/2
        │   ├── framing.go
        │   ├── streams.go
        │   └── hpack.go
        └── http_test.go        # Tests
```

---

### Core Types

```go
package http

type Request struct {
    Method      string
    URL         *url.URL
    Proto       string
    Header      Header
    Body        io.Reader
    Host        string
}

type Response struct {
    Status      string
    StatusCode  int
    Header      Header
    Body        io.Reader
}

type Client struct {
    Transport   RoundTripper
    Timeout     time.Duration
}
```

---

## Deliverables

- `pkg/http/` - HTTP implementation
- HTTP/1.1 client and server
- HTTP/2 support
- TLS integration
