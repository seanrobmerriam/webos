# PHASE 3.2: DNS Resolver

**Phase Context**: Phase 3 implements the networking stack. This sub-phase creates a DNS resolver without external dependencies.

**Sub-Phase Objective**: Implement DNS query/response parsing, recursive resolution, caching, and hosts file support.

**Prerequisites**: 
- Phase 3.1 (Network Stack) must be complete

**Integration Point**: DNS resolver will be used by HTTP client and network utilities.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a complete DNS client implementation from scratch.

---

### Directory Structure

```
webos/
└── pkg/
    └── net/
        └── dns/
            ├── doc.go          # Package documentation
            ├── dns.go          # DNS types and constants
            ├── resolver.go     # Resolver implementation
            ├── cache.go        # DNS cache
            ├── parser.go       # DNS message parsing
            └── dns_test.go     # Tests
```

---

### Core Types

```go
package dns

// Message represents a DNS message
type Message struct {
    ID           uint16
    QR           bool
    Opcode       Opcode
    AA           bool
    TC           bool
    RD           bool
    RA           bool
    Z            uint8
    RCODE        RCode
    Questions    []Question
    Answers      []ResourceRecord
}

// Resolver performs DNS lookups
type Resolver struct {
    servers     []string
    cache       *Cache
    timeout     time.Duration
    searchList  []string
}
```

---

### Implementation Steps

1. DNS message format parsing
2. Record types (A, AAAA, CNAME, MX, TXT, NS)
3. Query/response handling
4. TTL-based caching
5. Recursive resolution
6. Hosts file support
7. DNS-over-TLS

---

### Next Sub-Phase

**PHASE 3.3**: HTTP Client & Server

---

## Deliverables

- `pkg/net/dns/` - DNS resolver
- Query parsing and generation
- Caching system
- Hosts file support
