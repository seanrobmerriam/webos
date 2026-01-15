package process

import (
	"sync"
	"time"
)

// ProcessState represents the state of a process in the system.
type ProcessState string

const (
	// StateRunning indicates the process is currently executing.
	StateRunning ProcessState = "running"
	// StateWaiting indicates the process is blocked waiting for I/O or events.
	StateWaiting ProcessState = "waiting"
	// StateReady indicates the process is ready to run but waiting for CPU.
	StateReady ProcessState = "ready"
	// StateZombie indicates the process has terminated but parent hasn't collected status.
	StateZombie ProcessState = "zombie"
	// StateStopped indicates the process has been stopped (e.g., by a signal).
	StateStopped ProcessState = "stopped"
)

// Priority represents process scheduling priority.
type Priority int

const (
	// PriorityLow is the lowest priority level.
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Signal represents a signal number.
type Signal int

const (
	// SignalNil is a null signal (no operation).
	SignalNil Signal = 0
	// SignalInterrupt is sent when Ctrl+C is pressed.
	SignalInterrupt Signal = 2
	// SignalKill terminates the process immediately.
	SignalKill Signal = 9
	// SignalTerminate requests graceful termination.
	SignalTerminate Signal = 15
	// SignalChild is sent when a child process terminates.
	SignalChild Signal = 17
	// SignalStop stops the process.
	SignalStop Signal = 19
	// SignalContinue continues a stopped process.
	SignalContinue Signal = 18
)

// SignalSet represents a set of signals.
type SignalSet map[Signal]bool

// Has returns true if the signal set contains the given signal.
func (s SignalSet) Has(sig Signal) bool {
	_, ok := s[sig]
	return ok
}

// Add adds a signal to the set.
func (s SignalSet) Add(sig Signal) {
	s[sig] = true
}

// Remove removes a signal from the set.
func (s SignalSet) Remove(sig Signal) {
	delete(s, sig)
}

// Contains checks if a signal is in the set (alias for Has).
func (s SignalSet) Contains(sig Signal) bool {
	return s.Has(sig)
}

// IsEmpty returns true if the set is empty.
func (s SignalSet) IsEmpty() bool {
	return len(s) == 0
}

// Len returns the number of signals in the set.
func (s SignalSet) Len() int {
	return len(s)
}

// Union returns the union of two signal sets.
func (s SignalSet) Union(other SignalSet) SignalSet {
	result := make(SignalSet)
	for sig := range s {
		result[sig] = true
	}
	for sig := range other {
		result[sig] = true
	}
	return result
}

// Intersection returns the intersection of two signal sets.
func (s SignalSet) Intersection(other SignalSet) SignalSet {
	result := make(SignalSet)
	for sig := range s {
		if other[sig] {
			result[sig] = true
		}
	}
	return result
}

// Difference returns the difference of two signal sets.
func (s SignalSet) Difference(other SignalSet) SignalSet {
	result := make(SignalSet)
	for sig := range s {
		if !other[sig] {
			result[sig] = true
		}
	}
	return result
}

// NewSignalSet creates a new signal set with the given signals.
func NewSignalSet(signals ...Signal) SignalSet {
	ss := make(SignalSet)
	for _, sig := range signals {
		ss[sig] = true
	}
	return ss
}

// SignalHandler is a function that handles signals.
type SignalHandler func(p *Process, sig Signal)

// SignalHandlerMap maps signal numbers to handlers.
type SignalHandlerMap map[Signal]SignalHandler

// File represents an open file descriptor.
type File struct {
	// FD is the file descriptor number.
	FD int
	// Name is the file name.
	Name string
	// Mode is the file mode.
	Mode string
	// Offset is the current file offset.
	Offset int64
	// Data holds any buffered data.
	Data []byte
}

// NewFile creates a new file descriptor.
func NewFile(fd int, name string) *File {
	return &File{
		FD:     fd,
		Name:   name,
		Mode:   "rw",
		Offset: 0,
		Data:   nil,
	}
}

// ResourceLimits defines resource limits for a process.
type ResourceLimits struct {
	// CPUTime is the maximum CPU time allowed.
	CPUTime time.Duration
	// MaxMemory is the maximum memory usage in bytes.
	MaxMemory int64
	// MaxFiles is the maximum number of open files.
	MaxFiles int
	// MaxStackSize is the maximum stack size in bytes.
	MaxStackSize int64
	// MaxCoreSize is the maximum core dump size in bytes.
	MaxCoreSize int64
	// MaxDataSize is the maximum data segment size in bytes.
	MaxDataSize int64
	// MaxResidentSet is the maximum resident set size in bytes.
	MaxResidentSet int64
}

// DefaultLimits returns the default resource limits.
func DefaultLimits() *ResourceLimits {
	return &ResourceLimits{
		CPUTime:        time.Hour,
		MaxMemory:      512 * 1024 * 1024, // 512 MB
		MaxFiles:       1024,
		MaxStackSize:   8 * 1024 * 1024, // 8 MB
		MaxCoreSize:    0,               // No core dumps by default
		MaxDataSize:    0,               // Unlimited
		MaxResidentSet: 0,               // Unlimited
	}
}

// Process represents a virtual process in the system.
type Process struct {
	// PID is the unique process identifier.
	PID int
	// ParentPID is the PID of the parent process.
	ParentPID int
	// State is the current process state.
	State ProcessState
	// ExitCode is the process exit code (valid when state is Zombie).
	ExitCode int
	// CreatedAt is when the process was created.
	CreatedAt time.Time
	// StartedAt is when the process started executing.
	StartedAt time.Time
	// FinishedAt is when the process finished executing.
	FinishedAt time.Time
	// Command is the executable name or path.
	Command string
	// Args is the command-line arguments.
	Args []string
	// Env is the environment variables.
	Env []string
	// Cwd is the current working directory.
	Cwd string
	// Priority is the scheduling priority.
	Priority Priority
	// Limits defines resource limits for this process.
	Limits *ResourceLimits
	// Files holds open file descriptors.
	Files []*File
	// SignalMask defines which signals are blocked.
	SignalMask SignalSet
	// SignalHandlers defines custom signal handlers.
	SignalHandlers SignalHandlerMap
	// UserData holds arbitrary user-defined data.
	UserData map[string]interface{}
	// mu protects mutable process state.
	mu sync.Mutex
	// CPUUsage tracks CPU time consumed.
	CPUUsage time.Duration
	// MemoryUsage tracks current memory usage in bytes.
	MemoryUsage int64
}

// NewProcess creates a new process with the given configuration.
func NewProcess(pid int, parentPID int, command string, args []string) *Process {
	now := time.Now()
	return &Process{
		PID:            pid,
		ParentPID:      parentPID,
		State:          StateReady,
		Command:        command,
		Args:           args,
		Cwd:            "/",
		Priority:       PriorityNormal,
		CreatedAt:      now,
		StartedAt:      time.Time{},
		ExitCode:       0,
		Files:          make([]*File, 0),
		SignalMask:     make(SignalSet),
		SignalHandlers: make(SignalHandlerMap),
		UserData:       make(map[string]interface{}),
		CPUUsage:       0,
		MemoryUsage:    0,
	}
}

// SetState atomically sets the process state.
func (p *Process) SetState(state ProcessState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.State = state
}

// GetState atomically gets the process state.
func (p *Process) GetState() ProcessState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.State
}

// AddFile adds a file descriptor to the process.
func (p *Process) AddFile(f *File) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Files = append(p.Files, f)
}

// RemoveFile removes a file descriptor from the process.
func (p *Process) RemoveFile(fd int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fd >= 0 && fd < len(p.Files) {
		p.Files = append(p.Files[:fd], p.Files[fd+1:]...)
	}
}

// GetFile returns the file descriptor at the given index.
func (p *Process) GetFile(fd int) *File {
	p.mu.Lock()
	defer p.mu.Unlock()
	if fd >= 0 && fd < len(p.Files) {
		return p.Files[fd]
	}
	return nil
}

// FileCount returns the number of open files.
func (p *Process) FileCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.Files)
}

// SetEnvironment sets environment variables.
func (p *Process) SetEnvironment(env []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Env = env
}

// SetWorkingDirectory sets the current working directory.
func (p *Process) SetWorkingDirectory(cwd string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Cwd = cwd
}

// SetPriority sets the scheduling priority.
func (p *Process) SetPriority(priority Priority) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Priority = priority
}

// SetExitCode sets the process exit code.
func (p *Process) SetExitCode(code int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ExitCode = code
}

// AddCPUUsage adds to the CPU usage counter.
func (p *Process) AddCPUUsage(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CPUUsage += d
}

// SetMemoryUsage sets the current memory usage.
func (p *Process) SetMemoryUsage(bytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.MemoryUsage = bytes
}

// IsRunnable returns true if the process can be scheduled.
func (p *Process) IsRunnable() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.State == StateReady || p.State == StateRunning
}

// SetSignalHandler sets a handler for a specific signal.
func (p *Process) SetSignalHandler(sig Signal, handler SignalHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.SignalHandlers[sig] = handler
}

// HandleSignal processes a signal for this process.
func (p *Process) HandleSignal(sig Signal) {
	p.mu.Lock()
	blocked := p.SignalMask[sig]
	handler := p.SignalHandlers[sig]
	p.mu.Unlock()

	if blocked {
		return
	}

	if handler != nil {
		handler(p, sig)
	}
}

// SetUserData stores arbitrary user data.
func (p *Process) SetUserData(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.UserData[key] = value
}

// GetUserData retrieves user data by key.
func (p *Process) GetUserData(key string) (interface{}, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.UserData[key]
	return v, ok
}
