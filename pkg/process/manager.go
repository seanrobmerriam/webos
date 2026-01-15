package process

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Process creation errors.
var (
	ErrPIDInUse       = errors.New("PID already in use")
	ErrInvalidPID     = errors.New("invalid PID")
	ErrProcessExists  = errors.New("process already exists")
	ErrNotParent      = errors.New("not the parent process")
	ErrInvalidCommand = errors.New("invalid command")
)

// CreateConfig contains configuration for creating a new process.
type CreateConfig struct {
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
	// Limits defines resource limits.
	Limits *ResourceLimits
	// InheritFiles specifies whether to inherit parent's file descriptors.
	InheritFiles bool
}

// ProcessManager manages all processes in the system.
type ProcessManager struct {
	// processes holds all processes by PID.
	processes sync.Map
	// pidCounter generates unique PIDs.
	pidCounter int32
	// scheduler manages process scheduling.
	Scheduler Scheduler
	// enforcer manages resource limits.
	Enforcer *Enforcer
	// defaultLimits are applied to processes without specific limits.
	defaultLimits *ResourceLimits
	// mu protects global manager state.
	mu sync.RWMutex
	// children tracks parent-child relationships.
	children map[int][]int
	// zombieReaper reaps zombie processes.
	zombieReaper chan int
}

// NewProcessManager creates a new process manager.
func NewProcessManager(scheduler Scheduler) *ProcessManager {
	pm := &ProcessManager{
		pidCounter:    1,
		Scheduler:     scheduler,
		Enforcer:      NewEnforcer(),
		defaultLimits: DefaultLimits(),
		children:      make(map[int][]int),
		zombieReaper:  make(chan int, 100),
	}

	// Start zombie reaper goroutine
	go pm.reapZombies()

	return pm
}

// SetDefaultLimits sets the default resource limits for new processes.
func (pm *ProcessManager) SetDefaultLimits(limits *ResourceLimits) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.defaultLimits = limits
}

// GetDefaultLimits returns the current default resource limits.
func (pm *ProcessManager) GetDefaultLimits() *ResourceLimits {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.defaultLimits
}

// allocatePID allocates a new unique PID.
func (pm *ProcessManager) allocatePID() int {
	return int(atomic.AddInt32(&pm.pidCounter, 1))
}

// CreateProcess creates a new process with the given configuration.
func (pm *ProcessManager) CreateProcess(config *CreateConfig) (*Process, error) {
	if config.Command == "" {
		return nil, ErrInvalidCommand
	}

	pid := pm.allocatePID()

	// Check if PID is already in use
	if _, loaded := pm.processes.Load(pid); loaded {
		return nil, ErrPIDInUse
	}

	// Create the process
	p := NewProcess(pid, 0, config.Command, config.Args)

	// Set configuration
	if config.Env != nil {
		p.SetEnvironment(config.Env)
	}
	if config.Cwd != "" {
		p.SetWorkingDirectory(config.Cwd)
	}
	if config.Priority != 0 {
		p.SetPriority(config.Priority)
	}

	// Set limits
	if config.Limits != nil {
		p.Limits = config.Limits
	} else {
		p.Limits = pm.defaultLimits
	}

	// Register with enforcer
	pm.Enforcer.SetLimits(pid, p.Limits)

	// Store the process
	pm.processes.Store(pid, p)

	// Track in children map
	pm.mu.Lock()
	pm.children[0] = append(pm.children[0], pid)
	pm.mu.Unlock()

	return p, nil
}

// Fork creates a child process (copy of parent).
func (pm *ProcessManager) Fork(parent *Process, config *CreateConfig) (*Process, error) {
	if config.Command == "" {
		config.Command = parent.Command
	}
	if config.Args == nil {
		config.Args = parent.Args
	}

	child, err := pm.CreateProcess(config)
	if err != nil {
		return nil, err
	}

	// Inherit parent's working directory and environment
	child.Cwd = parent.Cwd
	child.Env = parent.Env

	// Track parent-child relationship
	pm.mu.Lock()
	pm.children[parent.PID] = append(pm.children[parent.PID], child.PID)
	child.ParentPID = parent.PID
	pm.mu.Unlock()

	return child, nil
}

// GetProcess retrieves a process by PID.
func (pm *ProcessManager) GetProcess(pid int) (*Process, error) {
	if pid <= 0 {
		return nil, ErrInvalidPID
	}

	p, ok := pm.processes.Load(pid)
	if !ok {
		return nil, ErrProcessNotFound
	}
	return p.(*Process), nil
}

// GetProcesses returns all processes.
func (pm *ProcessManager) GetProcesses() []*Process {
	processes := make([]*Process, 0)

	pm.processes.Range(func(key, value interface{}) bool {
		processes = append(processes, value.(*Process))
		return true
	})

	return processes
}

// GetChildren returns all child PIDs of a process.
func (pm *ProcessManager) GetChildren(pid int) []int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.children[pid]
}

// Start starts a process.
func (pm *ProcessManager) Start(pid int) error {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return err
	}

	if err := p.Start(); err != nil {
		return err
	}

	// Add to scheduler
	if pm.Scheduler != nil {
		pm.Scheduler.Schedule(p)
	}

	return nil
}

// Stop terminates a process with the given signal.
func (pm *ProcessManager) Stop(pid int, sig Signal) error {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return err
	}

	// Handle the signal
	p.HandleSignal(sig)

	// If kill signal, terminate immediately
	if sig == SignalKill {
		return pm.Terminate(pid, 128+int(sig))
	}

	return nil
}

// Terminate terminates a process with the given exit code.
func (pm *ProcessManager) Terminate(pid int, exitCode int) error {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return err
	}

	if err := p.Terminate(exitCode); err != nil {
		return err
	}

	// Signal parent about child termination
	if p.ParentPID > 0 {
		if parent, err := pm.GetProcess(p.ParentPID); err == nil {
			parent.HandleSignal(SignalChild)
		}
	}

	// Queue for reaping
	select {
	case pm.zombieReaper <- pid:
	default:
		// Channel full, process will be reaped later
	}

	return nil
}

// Wait waits for a process to terminate and returns its exit code.
func (pm *ProcessManager) Wait(pid int) (int, error) {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return 0, err
	}

	// Wait for process to become zombie
	for p.IsAlive() {
		// In a real implementation, this would use condition variables
	}

	return p.ExitCode, nil
}

// reapZombies is the zombie reaper goroutine.
func (pm *ProcessManager) reapZombies() {
	for pid := range pm.zombieReaper {
		pm.doReap(pid)
	}
}

// doReap performs the actual reaping of a zombie process.
func (pm *ProcessManager) doReap(pid int) {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return
	}

	// Only reap zombies
	if !p.IsTerminated() {
		return
	}

	// Remove from children tracking
	pm.mu.Lock()
	if children := pm.children[p.ParentPID]; len(children) > 0 {
		for i, child := range children {
			if child == pid {
				pm.children[p.ParentPID] = append(children[:i], children[i+1:]...)
				break
			}
		}
	}
	pm.mu.Unlock()

	// Remove from process table
	pm.processes.Delete(pid)

	// Clean up enforcer tracking
	pm.Enforcer.RemoveProcess(pid)
}

// Kill sends a kill signal to a process.
func (pm *ProcessManager) Kill(pid int) error {
	return pm.Stop(pid, SignalKill)
}

// Signal sends a signal to a process.
func (pm *ProcessManager) Signal(pid int, sig Signal) error {
	return pm.Stop(pid, sig)
}

// SetPriority changes the priority of a process.
func (pm *ProcessManager) SetPriority(pid int, priority Priority) error {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return err
	}

	p.SetPriority(priority)
	return nil
}

// GetCPUUsage returns CPU usage for a process.
func (pm *ProcessManager) GetCPUUsage(pid int) (time.Duration, error) {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return 0, err
	}

	usage, err := pm.Enforcer.GetUsage(pid)
	if err != nil {
		return p.CPUUsage, nil
	}

	usage.mu.Lock()
	defer usage.mu.Unlock()
	return usage.CPUTimeUsed, nil
}

// GetMemoryUsage returns memory usage for a process.
func (pm *ProcessManager) GetMemoryUsage(pid int) (int64, error) {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return 0, err
	}

	usage, err := pm.Enforcer.GetUsage(pid)
	if err != nil {
		return p.MemoryUsage, nil
	}

	usage.mu.Lock()
	defer usage.mu.Unlock()
	return usage.MemoryUsed, nil
}

// CountProcesses returns the total number of processes.
func (pm *ProcessManager) CountProcesses() int {
	count := 0
	pm.processes.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// FindProcessByCommand finds processes by command name.
func (pm *ProcessManager) FindProcessByCommand(command string) []*Process {
	processes := make([]*Process, 0)

	pm.processes.Range(func(key, value interface{}) bool {
		p := value.(*Process)
		if p.Command == command {
			processes = append(processes, p)
		}
		return true
	})

	return processes
}

// IsProcessAlive checks if a process is still alive.
func (pm *ProcessManager) IsProcessAlive(pid int) bool {
	p, err := pm.GetProcess(pid)
	if err != nil {
		return false
	}
	return p.IsAlive()
}
