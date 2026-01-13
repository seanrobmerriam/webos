# PHASE 5.1: Block Storage System

**Phase Context**: Phase 5 implements storage and data management. This sub-phase creates low-level storage abstraction.

**Sub-Phase Objective**: Implement block device abstraction, caching, write-ahead logging, RAID support, and snapshots.

**Prerequisites**: 
- Phase 2.1 (VFS) must be complete

**Integration Point**: Block storage provides the foundation for persistent file systems.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a complete block storage system with virtual devices, caching, and redundancy.

---

### Directory Structure

```
webos/
└── pkg/
    └── storage/
        ├── doc.go              # Package documentation
        ├── block.go            # Block device interface
        ├── cache.go            # Block caching
        ├── wal.go              # Write-ahead logging
        ├── raid.go             # RAID implementation
        ├── snapshot.go         # Snapshot management
        └── storage_test.go     # Tests
```

---

### Core Types

```go
package storage

type BlockDevice interface {
    Read(block uint64, data []byte) error
    Write(block uint64, data []byte) error
    BlockSize() int
    BlockCount() uint64
    Flush() error
    Close() error
}

type BlockCache struct {
    device    BlockDevice
    cache     map[uint64][]byte
    dirty     map[uint64]bool
    policy    CachePolicy // LRU, LFU
    maxSize   int
}

type RAID interface {
    Read(block uint64, data []byte) error
    Write(block uint64, data []byte) error
    Rebuild(failedDevice int) error
    Status() RAIDStatus
}
```

---

## Deliverables

- `pkg/storage/` - Block storage implementation
- Multiple caching policies
- RAID 0, 1, 5 support
- Snapshot system
- Encryption at rest
