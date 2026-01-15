package process

import (
	"errors"
	"sync"
	"time"
)

// Limit errors.
var (
	ErrLimitExceeded = errors.New("resource limit exceeded")
	ErrInvalidLimit  = errors.New("invalid resource limit value")
	ErrLimitNotSet   = errors.New("resource limit not set")
)

// ResourceType represents the type of resource being limited.
type ResourceType string

const (
	// ResourceCPU represents CPU time.
	ResourceCPU ResourceType = "cpu"
	// ResourceMemory represents memory usage.
	ResourceMemory ResourceType = "memory"
	// ResourceFiles represents open files.
	ResourceFiles ResourceType = "files"
	// ResourceStack represents stack size.
	ResourceStack ResourceType = "stack"
	// ResourceCore represents core dump size.
	ResourceCore ResourceType = "core"
	// ResourceData represents data segment size.
	ResourceData ResourceType = "data"
	// ResourceRSS represents resident set size.
	ResourceRSS ResourceType = "rss"
)

// ResourceUsage tracks current resource usage.
type ResourceUsage struct {
	// CPUTimeUsed is the CPU time used.
	CPUTimeUsed time.Duration
	// MemoryUsed is the current memory usage in bytes.
	MemoryUsed int64
	// FilesUsed is the number of open files.
	FilesUsed int
	// StackUsed is the stack size in bytes.
	StackUsed int64
	// CoreUsed is the core dump size in bytes.
	CoreUsed int64
	// DataUsed is the data segment size in bytes.
	DataUsed int64
	// RSSUsed is the resident set size in bytes.
	RSSUsed int64
	// mu protects the usage data.
	mu sync.Mutex
}

// NewResourceUsage creates a new resource usage tracker.
func NewResourceUsage() *ResourceUsage {
	return &ResourceUsage{
		CPUTimeUsed: 0,
		MemoryUsed:  0,
		FilesUsed:   0,
		StackUsed:   0,
		CoreUsed:    0,
		DataUsed:    0,
		RSSUsed:     0,
	}
}

// AddCPUTime adds to the CPU time used.
func (r *ResourceUsage) AddCPUTime(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.CPUTimeUsed += d
}

// SetMemory sets the current memory usage.
func (r *ResourceUsage) SetMemory(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.MemoryUsed = bytes
}

// AddFile increments the open file count.
func (r *ResourceUsage) AddFile() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.FilesUsed++
}

// RemoveFile decrements the open file count.
func (r *ResourceUsage) RemoveFile() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.FilesUsed > 0 {
		r.FilesUsed--
	}
}

// SetStack sets the stack size usage.
func (r *ResourceUsage) SetStack(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.StackUsed = bytes
}

// SetData sets the data segment size.
func (r *ResourceUsage) SetData(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.DataUsed = bytes
}

// SetRSS sets the resident set size.
func (r *ResourceUsage) SetRSS(bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.RSSUsed = bytes
}

// LimitChecker checks if resource usage is within limits.
type LimitChecker struct {
	limits *ResourceLimits
	usage  *ResourceUsage
}

// NewLimitChecker creates a new limit checker.
func NewLimitChecker(limits *ResourceLimits, usage *ResourceUsage) *LimitChecker {
	if limits == nil {
		limits = DefaultLimits()
	}
	if usage == nil {
		usage = NewResourceUsage()
	}
	return &LimitChecker{
		limits: limits,
		usage:  usage,
	}
}

// CheckCPU checks if CPU usage is within limits.
func (c *LimitChecker) CheckCPU() error {
	c.usage.mu.Lock()
	defer c.usage.mu.Unlock()

	if c.limits.CPUTime > 0 && c.usage.CPUTimeUsed >= c.limits.CPUTime {
		return &LimitError{
			Type:    ResourceCPU,
			Limit:   int64(c.limits.CPUTime),
			Used:    int64(c.usage.CPUTimeUsed),
			Message: "CPU time limit exceeded",
		}
	}
	return nil
}

// CheckMemory checks if memory usage is within limits.
func (c *LimitChecker) CheckMemory() error {
	c.usage.mu.Lock()
	defer c.usage.mu.Unlock()

	if c.limits.MaxMemory > 0 && c.usage.MemoryUsed >= c.limits.MaxMemory {
		return &LimitError{
			Type:    ResourceMemory,
			Limit:   c.limits.MaxMemory,
			Used:    c.usage.MemoryUsed,
			Message: "Memory limit exceeded",
		}
	}
	return nil
}

// CheckFiles checks if file count is within limits.
func (c *LimitChecker) CheckFiles() error {
	c.usage.mu.Lock()
	defer c.usage.mu.Unlock()

	if c.limits.MaxFiles > 0 && c.usage.FilesUsed >= c.limits.MaxFiles {
		return &LimitError{
			Type:    ResourceFiles,
			Limit:   int64(c.limits.MaxFiles),
			Used:    int64(c.usage.FilesUsed),
			Message: "File descriptor limit exceeded",
		}
	}
	return nil
}

// CheckStack checks if stack size is within limits.
func (c *LimitChecker) CheckStack() error {
	c.usage.mu.Lock()
	defer c.usage.mu.Unlock()

	if c.limits.MaxStackSize > 0 && c.usage.StackUsed >= c.limits.MaxStackSize {
		return &LimitError{
			Type:    ResourceStack,
			Limit:   c.limits.MaxStackSize,
			Used:    c.usage.StackUsed,
			Message: "Stack size limit exceeded",
		}
	}
	return nil
}

// CheckAll checks all resource limits.
func (c *LimitChecker) CheckAll() error {
	if err := c.CheckCPU(); err != nil {
		return err
	}
	if err := c.CheckMemory(); err != nil {
		return err
	}
	if err := c.CheckFiles(); err != nil {
		return err
	}
	if err := c.CheckStack(); err != nil {
		return err
	}
	return nil
}

// LimitError represents a resource limit violation.
type LimitError struct {
	Type    ResourceType
	Limit   int64
	Used    int64
	Message string
}

// Error returns the error message.
func (e *LimitError) Error() string {
	return e.Message
}

// IsLimitError checks if an error is a limit error.
func IsLimitError(err error) bool {
	_, ok := err.(*LimitError)
	return ok
}

// Enforcer manages resource limits for processes.
type Enforcer struct {
	limits map[int]*ResourceLimits
	usage  map[int]*ResourceUsage
	mu     sync.RWMutex
}

// NewEnforcer creates a new resource limit enforcer.
func NewEnforcer() *Enforcer {
	return &Enforcer{
		limits: make(map[int]*ResourceLimits),
		usage:  make(map[int]*ResourceUsage),
	}
}

// SetLimits sets resource limits for a process.
func (e *Enforcer) SetLimits(pid int, limits *ResourceLimits) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.limits[pid] = limits
	e.usage[pid] = NewResourceUsage()
}

// GetLimits gets resource limits for a process.
func (e *Enforcer) GetLimits(pid int) (*ResourceLimits, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	limits, ok := e.limits[pid]
	if !ok {
		return nil, ErrLimitNotSet
	}
	return limits, nil
}

// GetUsage gets resource usage for a process.
func (e *Enforcer) GetUsage(pid int) (*ResourceUsage, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	usage, ok := e.usage[pid]
	if !ok {
		return nil, ErrLimitNotSet
	}
	return usage, nil
}

// CheckLimits checks if a process is within its resource limits.
func (e *Enforcer) CheckLimits(pid int) error {
	e.mu.RLock()
	limits, ok := e.limits[pid]
	usage, ok2 := e.usage[pid]
	e.mu.RUnlock()

	if !ok || !ok2 {
		return ErrLimitNotSet
	}

	checker := NewLimitChecker(limits, usage)
	return checker.CheckAll()
}

// AddFileUsage increments the file count for a process.
func (e *Enforcer) AddFileUsage(pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	usage, ok := e.usage[pid]
	if !ok {
		return ErrLimitNotSet
	}

	limits, ok := e.limits[pid]
	if !ok {
		limits = DefaultLimits()
	}

	if limits.MaxFiles > 0 && usage.FilesUsed >= limits.MaxFiles {
		return &LimitError{
			Type:    ResourceFiles,
			Limit:   int64(limits.MaxFiles),
			Used:    int64(usage.FilesUsed),
			Message: "File descriptor limit exceeded",
		}
	}

	usage.AddFile()
	return nil
}

// RemoveFileUsage decrements the file count for a process.
func (e *Enforcer) RemoveFileUsage(pid int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if usage, ok := e.usage[pid]; ok {
		usage.RemoveFile()
	}
}

// UpdateMemoryUsage updates memory usage for a process.
func (e *Enforcer) UpdateMemoryUsage(pid int, bytes int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	usage, ok := e.usage[pid]
	if !ok {
		return ErrLimitNotSet
	}

	limits, ok := e.limits[pid]
	if !ok {
		limits = DefaultLimits()
	}

	usage.SetMemory(bytes)

	if limits.MaxMemory > 0 && bytes >= limits.MaxMemory {
		return &LimitError{
			Type:    ResourceMemory,
			Limit:   limits.MaxMemory,
			Used:    bytes,
			Message: "Memory limit exceeded",
		}
	}

	return nil
}

// UpdateCPUUsage updates CPU time for a process.
func (e *Enforcer) UpdateCPUUsage(pid int, d time.Duration) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	usage, ok := e.usage[pid]
	if !ok {
		return ErrLimitNotSet
	}

	limits, ok := e.limits[pid]
	if !ok {
		limits = DefaultLimits()
	}

	usage.AddCPUTime(d)

	if limits.CPUTime > 0 && usage.CPUTimeUsed >= limits.CPUTime {
		return &LimitError{
			Type:    ResourceCPU,
			Limit:   int64(limits.CPUTime),
			Used:    int64(usage.CPUTimeUsed),
			Message: "CPU time limit exceeded",
		}
	}

	return nil
}

// RemoveProcess removes process limits and usage tracking.
func (e *Enforcer) RemoveProcess(pid int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.limits, pid)
	delete(e.usage, pid)
}
