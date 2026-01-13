# PHASE 2.2: Process Management System

**Phase Context**: Phase 2 implements core system utilities. This sub-phase creates the process management system for virtual processes.

**Sub-Phase Objective**: Implement process lifecycle management, scheduler with priority-based scheduling, IPC mechanisms, and resource limits.

**Prerequisites**: 
- Phase 2.1 (VFS) recommended

**Integration Point**: Process management will be used by shell, system utilities, and background services.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a process management system with cooperative multitasking, inspired by Unix process model.

---

### Directory Structure

```
webos/
├── pkg/
│   └── process/
│       ├── doc.go              # Package documentation
│       ├── process.go          # Process structure
│       ├── manager.go          # Process lifecycle
│       ├── scheduler.go        # Task scheduling
│       ├── state.go            # Process state machine
│       ├── limits.go           # Resource limits
│       ├── ipc/                # IPC mechanisms
│       │   ├── pipe.go         # Anonymous pipes
│       │   ├── namedpipe.go    # Named pipes (FIFO)
│       │   ├── message.go      # Message queues
│       │   ├── sharedmem.go    # Shared memory
│       │   └── signal.go       # Signal handling
│       └── process_test.go     # Tests
└── cmd/
    └── process-demo/
        └── main.go             # Demonstration program
```

---

### Core Types

```go
package process

// ProcessState represents process state
type ProcessState string

const (
    StateRunning    ProcessState = "running"
    StateWaiting    ProcessState = "waiting"
    StateReady      ProcessState = "ready"
    StateZombie     ProcessState = "zombie"
    StateStopped    ProcessState = "stopped"
)

// Process represents a virtual process
type Process struct {
    PID         int
    ParentPID   int
    State       ProcessState
    ExitCode    int
    CreatedAt   time.Time
    StartedAt   time.Time
    Command     string
    Args        []string
    Env         []string
    Cwd         string
    Capabilities security.Promise
    UnveilPaths []security.UnveilPath
    Limits      *ResourceLimits
    Files       []*File
    SignalMask  SignalSet
}

// ProcessManager manages all processes
type ProcessManager struct {
    processes sync.Map
    pidCounter int32
    scheduler  Scheduler
}

// Scheduler implements process scheduling
type Scheduler interface {
    Schedule(p *Process) error
    Yield()
    GetNextRunnable() *Process
}
```

---

### Testing Requirements

- Process creation and termination
- Scheduler fairness
- Resource limit enforcement
- IPC message passing
- Signal delivery

---

### Next Sub-Phase

**PHASE 2.3**: Shell Implementation

---

## Deliverables

- `pkg/process/` - Complete process management
- Scheduler implementation
- IPC primitives
- Resource limiting
- Comprehensive tests
