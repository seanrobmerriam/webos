# WebOS Project Agent Prompts

This directory contains agent prompts for each sub-phase of the WebOS project development.

## Project Overview

WebOS is a web-based operating system inspired by OpenBSD's security philosophy and implemented using the Guacamole architecture pattern. The project is written entirely in Go for backend services and vanilla JavaScript for the browser client, with zero external dependencies.

## Architecture

```
┌─────────────────────────────────────┐
│     Web Client (Browser)            │
│  - Vanilla JavaScript UI Layer      │
│  - Custom Protocol Client           │
│  - Canvas Rendering Engine          │
└──────────────┬──────────────────────┘
               │ WebSocket/HTTP
               │ (Custom Protocol)
┌──────────────┴──────────────────────┐
│   Web Application Layer (Go)        │
│  - Authentication/Authorization     │
│  - Session Management               │
│  - Protocol Translation Hub         │
└──────────────┬──────────────────────┘
               │ Internal Protocol
┌──────────────┴──────────────────────┐
│   Backend Services Daemon (Go)      │
│  - Process Management               │
│  - File System Operations           │
│  - Network Stack                    │
│  - Security Enforcement             │
└─────────────────────────────────────┘
```

## Phases Overview

| Phase | Name | Duration | Focus |
|-------|------|----------|-------|
| 1 | Foundation & Core Protocol | Months 1-6 | Communication layer, security |
| 2 | Core System Utilities | Months 7-12 | VFS, processes, shell, terminal |
| 3 | Networking Stack | Months 13-18 | TCP/IP, HTTP, firewall, services |
| 4 | User Interface & Window System | Months 19-24 | Window manager, graphics, widgets |
| 5 | Storage & Data Management | Months 25-30 | Block storage, database, KV store |

## Phase Prompts

### Phase 1: Foundation & Core Protocol (Months 1-6)

**Objective**: Establish the fundamental architecture, protocol, and basic communication layer.

| Sub-Phase | Prompt | Description |
|-----------|--------|-------------|
| 1.1 | [phase-1-1-custom-protocol.md](phase-1-1-custom-protocol.md) | Custom binary protocol design and implementation |
| 1.2 | [phase-1-2-websocket-server.md](phase-1-2-websocket-server.md) | WebSocket server and connection management |
| 1.3 | [phase-1-3-security-foundation.md](phase-1-3-security-foundation.md) | OpenBSD-inspired security model |
| 1.4 | [phase-1-4-http-server.md](phase-1-4-http-server.md) | HTTP server and routing |
| 1.5 | [phase-1-5-client-foundation.md](phase-1-5-client-foundation.md) | Browser-based client foundation |

**Milestones**:
- M1.1: Protocol specification complete, implementations working
- M1.2: WebSocket server operational
- M1.3: Security foundation complete
- M1.4: HTTP server serving client
- M1.5: Client can connect and display basic interface

---

### Phase 2: Core System Utilities (Months 7-12)

**Objective**: Implement essential operating system utilities following OpenBSD's philosophy.

| Sub-Phase | Prompt | Description |
|-----------|--------|-------------|
| 2.1 | [phase-2-1-virtual-filesystem.md](phase-2-1-virtual-filesystem.md) | Virtual file system with multiple backends |
| 2.2 | [phase-2-2-process-management.md](phase-2-2-process-management.md) | Process management and IPC |
| 2.3 | [phase-2-3-shell-implementation.md](phase-2-3-shell-implementation.md) | POSIX-compliant shell (wsh) |
| 2.4 | [phase-2-4-terminal-emulator.md](phase-2-4-terminal-emulator.md) | VT100/xterm terminal emulation |
| 2.5 | [phase-2-5-core-utilities.md](phase-2-5-core-utilities.md) | 30+ core command-line utilities |

**Milestones**:
- M2.1: VFS complete, file operations working
- M2.2: Process manager operational
- M2.3: Shell complete
- M2.4: Terminal emulator working
- M2.5: Core utilities complete

---

### Phase 3: Networking Stack (Months 13-18)

**Objective**: Implement a complete TCP/IP networking stack from scratch.

| Sub-Phase | Prompt | Description |
|-----------|--------|-------------|
| 3.1 | [phase-3-1-network-stack.md](phase-3-1-network-stack.md) | TCP/IP stack (Ethernet, IP, TCP, UDP, ICMP) |
| 3.2 | [phase-3-2-dns-resolver.md](phase-3-2-dns-resolver.md) | DNS resolver |
| 3.3 | [phase-3-3-http-implementation.md](phase-3-3-http-implementation.md) | HTTP/1.1 and HTTP/2 |
| 3.4 | [phase-3-4-firewall.md](phase-3-4-firewall.md) | OpenBSD PF-inspired firewall |
| 3.5 | [phase-3-5-network-services.md](phase-3-5-network-services.md) | SSH, HTTP, FTP servers |

**Milestones**:
- M3.1: TCP/IP stack complete
- M3.2: DNS resolution functional
- M3.3: HTTP client/server working
- M3.4: Firewall operational
- M3.5: Network services running

---

### Phase 4: User Interface & Window System (Months 19-24)

**Objective**: Implement a complete windowing system and graphical user interface.

| Sub-Phase | Prompt | Description |
|-----------|--------|-------------|
| 4.1 | [phase-4-1-window-manager.md](phase-4-1-window-manager.md) | Multi-window management |
| 4.2 | [phase-4-2-graphics-library.md](phase-4-2-graphics-library.md) | 2D graphics primitives |
| 4.3 | [phase-4-3-widget-toolkit.md](phase-4-3-widget-toolkit.md) | Standard UI components |
| 4.4 | [phase-4-4-desktop-environment.md](phase-4-4-desktop-environment.md) | Complete desktop environment |

**Milestones**:
- M4.1: Window manager operational
- M4.2: Graphics library complete
- M4.3: Widget toolkit ready
- M4.4: Desktop environment functional

---

### Phase 5: Storage & Data Management (Months 25-30)

**Objective**: Implement persistent storage, database systems, and data management.

| Sub-Phase | Prompt | Description |
|-----------|--------|-------------|
| 5.1 | [phase-5-1-block-storage.md](phase-5-1-block-storage.md) | Block storage system |
| 5.2 | [phase-5-2-database-engine.md](phase-5-2-database-engine.md) | SQL database engine |
| 5.3 | [phase-5-3-key-value-store.md](phase-5-3-key-value-store.md) | LSM-tree key-value store |
| 5.4 | [phase-5-4-data-synchronization.md](phase-5-4-data-synchronization.md) | Backup and sync |

**Milestones**:
- M5.1: Block storage operational
- M5.2: Database engine functional
- M5.3: KV store complete
- M5.4: Backup and sync working

---

## Quick Start

To begin development, start with **Phase 1.1** and work through each sub-phase sequentially. Each prompt contains:

1. **Phase Context**: How this phase fits into the overall project
2. **Sub-Phase Objective**: What this sub-phase accomplishes
3. **Prerequisites**: What must be complete before starting
4. **Directory Structure**: Where to create files
5. **Core Types**: Key data structures and interfaces
6. **Implementation Steps**: Detailed step-by-step instructions
7. **Testing Requirements**: What tests to write
8. **Deliverables**: Expected output files

## Validation Commands

After completing each sub-phase, run:

```bash
# All tests pass
go test ./... -v

# No race conditions
go test ./... -race

# No vet warnings
go vet ./...

# All demos build
go build ./cmd/...

# Coverage check
go test ./... -cover
```

## Design Principles

All phases follow these OpenBSD-inspired principles:

1. **Security by Design**: Proactive security from day one
2. **Correctness Over Features**: Quality prioritized over velocity
3. **Integrated Base System**: Coherent, unified system
4. **Modularity**: Clean separation of concerns
5. **Zero Dependencies**: Pure Go stdlib only

## Technology Stack

### Backend (Go)
- Language: Go 1.21+
- Networking: `net/http`, `net`, `crypto/tls`
- Concurrency: Goroutines and channels
- Storage: Custom file format parsers

### Frontend (JavaScript)
- Language: Modern JavaScript (ES6+)
- Graphics: HTML5 Canvas API
- Communication: WebSocket API
- Storage: IndexedDB (browser-native)
- **No frameworks or libraries**

## Next Steps

1. Review the [DEVPLAN.md](../DEVPLAN.md) for full context
2. Read the [PROJECT.md](../PROJECT.md) for project specifics
3. Start with **Phase 1.1**: [phase-1-1-custom-protocol.md](phase-1-1-custom-protocol.md)
4. Follow the prompt sequentially through each sub-phase

## Questions?

Refer to:
- [TEMPLATE.md](TEMPLATE.md) - Template for understanding prompt structure
- [AGENTS.md](../AGENTS.md) - Agent rules and standards
- [DEVPLAN.md](../DEVPLAN.md) - Full development plan
- [PROJECT.md](../PROJECT.md) - Project specifications
