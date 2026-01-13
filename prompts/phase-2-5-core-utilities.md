# PHASE 2.5: Core System Utilities

**Phase Context**: Phase 2 implements core system utilities. This sub-phase creates essential command-line utilities.

**Sub-Phase Objective**: Implement 30+ core utilities including file operations, text processing, system information, and network utilities.

**Prerequisites**: 
- Phase 2.1 (VFS) must be complete
- Phase 2.2 (Process) recommended

**Integration Point**: Utilities are executed via shell and use VFS/process management.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing essential command-line utilities from scratch, following OpenBSD's philosophy of correctness.

---

### Directory Structure

```
webos/
├── cmd/
│   ├── utils/
│   │   ├── file/               # File operations
│   │   │   ├── ls.go, cat.go, cp.go, mv.go, rm.go
│   │   │   ├── mkdir.go, touch.go, chmod.go, chown.go
│   │   ├── text/               # Text processing
│   │   │   ├── grep.go, sed.go, awk.go, cut.go
│   │   │   ├── sort.go, uniq.go, wc.go, head.go, tail.go
│   │   ├── system/             # System information
│   │   │   ├── ps.go, top.go, df.go, du.go
│   │   │   ├── uname.go, date.go, uptime.go, whoami.go, env.go
│   │   └── network/            # Network utilities
│   │       ├── ping.go, netcat.go, curl.go, wget.go
│   └── utils-demo/
│       └── main.go             # Demonstration
```

---

### Utility Specifications

**File Operations**: `ls`, `cat`, `cp`, `mv`, `rm`, `mkdir`, `touch`, `chmod`, `chown`

**Text Processing**: `grep`, `sed`, `awk`, `cut`, `sort`, `uniq`, `wc`, `head`, `tail`

**System Information**: `ps`, `top`, `df`, `du`, `uname`, `date`, `uptime`, `whoami`, `env`

**Network Utilities**: `ping`, `netcat`, `curl`, `wget`

---

### Implementation Requirements

1. Consistent flag parsing
2. POSIX-compatible behavior
3. Proper error handling
4. Man pages for each utility
5. Integration with shell
6. Test suite for each utility

---

## Deliverables

- 30+ core utilities in `cmd/utils/`
- Consistent flag parsing
- Man pages for each utility
- Comprehensive test suites
