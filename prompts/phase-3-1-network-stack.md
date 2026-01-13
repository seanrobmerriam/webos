# PHASE 3.1: Network Protocol Stack

**Phase Context**: Phase 3 implements the networking stack. This sub-phase creates a complete TCP/IP implementation from scratch.

**Sub-Phase Objective**: Implement OSI Layer 2-4 (Ethernet, IP, TCP, UDP, ICMP) with routing, fragmentation, and congestion control.

**Prerequisites**: 
- Phase 2 complete

**Integration Point**: Network stack will be used by all network services and utilities.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a complete TCP/IP stack from scratch using Go, including Ethernet, IP, TCP, UDP, and ICMP.

---

### Directory Structure

```
webos/
├── pkg/
│   └── net/
│       ├── doc.go              # Package documentation
│       ├── interface.go        # Network interface abstraction
│       ├── ethernet/           # Layer 2
│       │   ├── ethernet.go
│       │   ├── arp.go
│       │   └── ethernet_test.go
│       ├── ip/                 # Layer 3
│       │   ├── ipv4.go
│       │   ├── ipv6.go
│       │   ├── icmp.go
│       │   ├── fragmentation.go
│       │   └── routing.go
│       ├── tcp/                # Layer 4
│       │   ├── tcp.go
│       │   ├── connection.go
│       │   ├── sliding_window.go
│       │   ├── congestion.go
│       │   └── tcp_test.go
│       ├── udp/                # Layer 4
│       │   ├── udp.go
│       │   └── udp_test.go
│       └── socket/             # Socket API
│           ├── socket.go
│           └── socket_test.go
└── cmd/
    └── netstack-demo/
        └── main.go             # Demonstration program
```

---

### Core Types

```go
package net

// Packet represents a network packet
type Packet struct {
    Data       []byte
    SrcIP      net.IP
    DstIP      net.IP
    Protocol   Protocol
    Iface      *Interface
    Timestamp  time.Time
}

// Interface represents a network interface
type Interface struct {
    Name        string
    MAC         net.HardwareAddr
    IP          net.IP
    Mask        net.IPMask
    MTU         int
    Flags       InterfaceFlags
}

// IPPacket represents an IP packet
type IPPacket struct {
    Version     uint8
    TOS         uint8
    Length      uint16
    ID          uint16
    Flags       uint8
    FragOffset  uint16
    TTL         uint8
    Protocol    uint8
    Checksum    uint16
    SrcIP       net.IP
    DstIP       net.IP
    Payload     []byte
}

// TCPSegment represents a TCP segment
type TCPSegment struct {
    SrcPort     uint16
    DstPort     uint16
    SeqNum      uint32
    AckNum      uint32
    DataOffset  uint8
    Flags       TCPFlags
    WindowSize  uint16
    Checksum    uint16
    Urgent      uint16
    Payload     []byte
}
```

---

### Implementation Steps

1. **Ethernet Layer**: Frame parsing, MAC addressing, ARP protocol
2. **IP Layer**: IPv4/IPv6 implementation, fragmentation/reassembly
3. **ICMP**: Ping and traceroute support
4. **TCP**: RFC 793 compliance, sliding window, congestion control
5. **UDP**: Datagram handling, port multiplexing
6. **Routing**: Routing table, packet forwarding
7. **Socket API**: BSD-socket-like interface

---

### Testing Requirements

- Protocol conformance tests
- Packet parsing/generation
- TCP state machine transitions
- Congestion control behavior
- Fragmentation/reassembly

---

### Next Sub-Phase

**PHASE 3.2**: DNS Resolver

---

## Deliverables

- `pkg/net/` - Complete network stack
- Ethernet, IP, TCP, UDP, ICMP implementations
- Routing table
- Socket API
- Protocol conformance tests
