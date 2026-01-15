package process

import (
	"testing"
	"time"
)

// TestProcessStateTransitions tests valid state transitions.
func TestProcessStateTransitions(t *testing.T) {
	p := NewProcess(1, 0, "test", []string{"arg1"})

	// Test valid transitions
	tests := []struct {
		name    string
		from    ProcessState
		to      ProcessState
		wantErr bool
	}{
		{"Ready to Running", StateReady, StateRunning, false},
		{"Running to Waiting", StateRunning, StateWaiting, false},
		{"Waiting to Ready", StateWaiting, StateReady, false},
		{"Running to Ready", StateRunning, StateReady, false},
		{"Running to Zombie", StateRunning, StateZombie, false},
		{"Waiting to Zombie", StateWaiting, StateZombie, false},
		{"Running to Stopped", StateRunning, StateStopped, false},
		{"Stopped to Ready", StateStopped, StateReady, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p.State = tt.from
			err := p.TransitionTo(tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test invalid transitions
	invalidTransitions := []struct {
		name string
		from ProcessState
		to   ProcessState
	}{
		{"Ready to Zombie", StateReady, StateZombie},
		{"Zombie to Running", StateZombie, StateRunning},
		{"Stopped to Zombie", StateStopped, StateZombie},
	}

	for _, tt := range invalidTransitions {
		t.Run(tt.name, func(t *testing.T) {
			p.State = tt.from
			err := p.TransitionTo(tt.to)
			if err == nil {
				t.Errorf("TransitionTo() should fail for invalid transition from %s to %s", tt.from, tt.to)
			}
		})
	}
}

// TestProcessMethods tests process methods.
func TestProcessMethods(t *testing.T) {
	p := NewProcess(1, 0, "test", []string{"arg1"})

	// Test SetState and GetState
	p.SetState(StateRunning)
	if p.GetState() != StateRunning {
		t.Errorf("GetState() = %v, want %v", p.GetState(), StateRunning)
	}

	// Test AddFile and GetFile
	f := NewFile(0, "stdout")
	p.AddFile(f)
	if p.GetFile(0) != f {
		t.Errorf("GetFile(0) = %v, want %v", p.GetFile(0), f)
	}
	if p.FileCount() != 1 {
		t.Errorf("FileCount() = %d, want 1", p.FileCount())
	}

	// Test SetPriority
	p.SetPriority(PriorityHigh)
	if p.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", p.Priority, PriorityHigh)
	}

	// Test SetExitCode
	p.SetExitCode(42)
	if p.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", p.ExitCode)
	}

	// Test IsRunnable
	if !p.IsRunnable() {
		t.Error("IsRunnable() = false, want true for StateRunning")
	}
}

// TestProcessManagerCreate tests process creation.
func TestProcessManagerCreate(t *testing.T) {
	scheduler := NewPriorityScheduler()
	pm := NewProcessManager(scheduler)

	config := &CreateConfig{
		Command:  "test",
		Args:     []string{"arg1", "arg2"},
		Env:      []string{"PATH=/bin"},
		Cwd:      "/",
		Priority: PriorityNormal,
	}

	p, err := pm.CreateProcess(config)
	if err != nil {
		t.Fatalf("CreateProcess() error = %v", err)
	}

	if p.Command != "test" {
		t.Errorf("Command = %s, want test", p.Command)
	}
	if p.PID <= 0 {
		t.Errorf("PID = %d, want > 0", p.PID)
	}
	if p.State != StateReady {
		t.Errorf("State = %v, want %v", p.State, StateReady)
	}
}

// TestProcessManagerStartStop tests starting and stopping processes.
func TestProcessManagerStartStop(t *testing.T) {
	scheduler := NewPriorityScheduler()
	pm := NewProcessManager(scheduler)

	config := &CreateConfig{
		Command: "test",
	}
	p, err := pm.CreateProcess(config)
	if err != nil {
		t.Fatalf("CreateProcess() error = %v", err)
	}

	// Start the process
	err = pm.Start(p.PID)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if p.GetState() != StateRunning {
		t.Errorf("State = %v, want %v", p.GetState(), StateRunning)
	}

	// Stop the process
	err = pm.Terminate(p.PID, 0)
	if err != nil {
		t.Fatalf("Terminate() error = %v", err)
	}

	if p.GetState() != StateZombie {
		t.Errorf("State = %v, want %v", p.GetState(), StateZombie)
	}
}

// TestSchedulerPriorityQueue tests the priority queue.
func TestSchedulerPriorityQueue(t *testing.T) {
	q := NewProcessQueue()

	p1 := NewProcess(1, 0, "test1", nil)
	p2 := NewProcess(2, 0, "test2", nil)

	// Push processes
	q.Push(p1)
	q.Push(p2)

	if q.Len() != 2 {
		t.Errorf("Len() = %d, want 2", q.Len())
	}

	// Check Contains
	if !q.Contains(p1.PID) {
		t.Error("Contains(1) = false, want true")
	}

	// Remove
	q.Remove(p1.PID)
	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}
}

// TestSchedulerSchedule tests the priority scheduler.
func TestSchedulerSchedule(t *testing.T) {
	scheduler := NewPriorityScheduler()

	p1 := NewProcess(1, 0, "test1", nil)
	p1.SetPriority(PriorityLow)
	p2 := NewProcess(2, 0, "test2", nil)
	p2.SetPriority(PriorityHigh)

	// Schedule processes
	scheduler.Schedule(p1)
	scheduler.Schedule(p2)

	if scheduler.Len() != 2 {
		t.Errorf("Len() = %d, want 2", scheduler.Len())
	}

	// Get next should return p2 (higher priority)
	next := scheduler.GetNextRunnable()
	if next.PID != 2 {
		t.Errorf("GetNextRunnable() = PID %d, want 2 (higher priority)", next.PID)
	}

	// The remaining should be p1
	next2 := scheduler.GetNextRunnable()
	if next2.PID != 1 {
		t.Errorf("GetNextRunnable() = PID %d, want 1 (remaining)", next2.PID)
	}
}

// TestResourceLimits tests resource limit checking.
func TestResourceLimits(t *testing.T) {
	limits := DefaultLimits()
	usage := NewResourceUsage()
	checker := NewLimitChecker(limits, usage)

	// Initially should pass
	if err := checker.CheckAll(); err != nil {
		t.Errorf("CheckAll() error = %v", err)
	}

	// Add file usage
	usage.AddFile()
	usage.AddFile()
	if err := checker.CheckFiles(); err != nil {
		t.Errorf("CheckFiles() error = %v", err)
	}
}

// TestResourceLimitsExceeded tests limit exceeded errors.
func TestResourceLimitsExceeded(t *testing.T) {
	limits := &ResourceLimits{
		MaxMemory: 100,
		MaxFiles:  2,
	}
	usage := NewResourceUsage()
	checker := NewLimitChecker(limits, usage)

	// Exceed file limit
	usage.AddFile()
	usage.AddFile()
	usage.AddFile()

	err := checker.CheckFiles()
	if err == nil {
		t.Error("CheckFiles() should return error when limit exceeded")
	}
}

// TestEnforcer tests the resource enforcer.
func TestEnforcer(t *testing.T) {
	enforcer := NewEnforcer()

	// Set limits
	enforcer.SetLimits(1, &ResourceLimits{MaxFiles: 5})

	// Add file usage
	err := enforcer.AddFileUsage(1)
	if err != nil {
		t.Errorf("AddFileUsage() error = %v", err)
	}

	// Check limits
	err = enforcer.CheckLimits(1)
	if err != nil {
		t.Errorf("CheckLimits() error = %v", err)
	}

	// Remove process
	enforcer.RemoveProcess(1)
}

// TestSignalSet tests signal set operations.
func TestSignalSet(t *testing.T) {
	ss := make(SignalSet)

	// Add signals
	ss.Add(SignalInterrupt)
	ss.Add(SignalKill)

	if !ss.Contains(SignalInterrupt) {
		t.Error("Contains(SignalInterrupt) = false, want true")
	}
	if !ss.Contains(SignalKill) {
		t.Error("Contains(SignalKill) = false, want true")
	}

	// Remove signal
	ss.Remove(SignalInterrupt)
	if ss.Contains(SignalInterrupt) {
		t.Error("Contains(SignalInterrupt) = true, want false after removal")
	}

	// Union
	other := NewSignalSet(SignalTerminate, SignalChild)
	union := ss.Union(other)

	if !union.Contains(SignalKill) {
		t.Error("Union should contain SignalKill")
	}
	if !union.Contains(SignalTerminate) {
		t.Error("Union should contain SignalTerminate")
	}

	// Intersection
	intersection := ss.Intersection(other)
	if !intersection.IsEmpty() {
		t.Error("Intersection of disjoint sets should be empty")
	}
}

// TestPriorityLevels tests priority levels.
func TestPriorityLevels(t *testing.T) {
	if PriorityLow >= PriorityNormal {
		t.Error("PriorityLow should be < PriorityNormal")
	}
	if PriorityNormal >= PriorityHigh {
		t.Error("PriorityNormal should be < PriorityHigh")
	}
	if PriorityHigh >= PriorityCritical {
		t.Error("PriorityHigh should be < PriorityCritical")
	}
}

// TestProcessLifetime tests process lifetime calculations.
func TestProcessLifetime(t *testing.T) {
	p := NewProcess(1, 0, "test", nil)

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	lifetime := p.TotalLifetime()
	if lifetime < 10*time.Millisecond {
		t.Errorf("TotalLifetime() = %v, want >= 10ms", lifetime)
	}
}

// TestForkProcess tests forking a process.
func TestForkProcess(t *testing.T) {
	scheduler := NewPriorityScheduler()
	pm := NewProcessManager(scheduler)

	// Create parent
	parent, err := pm.CreateProcess(&CreateConfig{
		Command: "parent",
		Args:    []string{"--parent"},
		Cwd:     "/tmp",
	})
	if err != nil {
		t.Fatalf("CreateProcess() error = %v", err)
	}

	// Fork child
	child, err := pm.Fork(parent, &CreateConfig{})
	if err != nil {
		t.Fatalf("Fork() error = %v", err)
	}

	if child.ParentPID != parent.PID {
		t.Errorf("Child.ParentPID = %d, want %d", child.ParentPID, parent.PID)
	}

	if child.Cwd != parent.Cwd {
		t.Errorf("Child.Cwd = %s, want %s", child.Cwd, parent.Cwd)
	}

	// Check children tracking
	children := pm.GetChildren(parent.PID)
	if len(children) != 1 || children[0] != child.PID {
		t.Errorf("GetChildren() = %v, want [%d]", children, child.PID)
	}
}

// TestFindProcessByCommand tests finding processes by command.
func TestFindProcessByCommand(t *testing.T) {
	scheduler := NewPriorityScheduler()
	pm := NewProcessManager(scheduler)

	// Create processes
	pm.CreateProcess(&CreateConfig{Command: "echo"})
	pm.CreateProcess(&CreateConfig{Command: "ls"})
	pm.CreateProcess(&CreateConfig{Command: "echo"})

	// Find echo processes
	processes := pm.FindProcessByCommand("echo")
	if len(processes) != 2 {
		t.Errorf("FindProcessByCommand() returned %d processes, want 2", len(processes))
	}
}

// TestProcessManagerCount tests counting processes.
func TestProcessManagerCount(t *testing.T) {
	scheduler := NewPriorityScheduler()
	pm := NewProcessManager(scheduler)

	if pm.CountProcesses() != 0 {
		t.Errorf("CountProcesses() = %d, want 0", pm.CountProcesses())
	}

	pm.CreateProcess(&CreateConfig{Command: "test1"})
	pm.CreateProcess(&CreateConfig{Command: "test2"})

	if pm.CountProcesses() != 2 {
		t.Errorf("CountProcesses() = %d, want 2", pm.CountProcesses())
	}
}

// TestProcessSignalHandling tests signal handling.
func TestProcessSignalHandling(t *testing.T) {
	p := NewProcess(1, 0, "test", nil)

	received := false
	p.SetSignalHandler(SignalInterrupt, func(proc *Process, sig Signal) {
		received = true
	})

	p.HandleSignal(SignalInterrupt)
	if !received {
		t.Error("Signal handler was not called")
	}
}

// TestBlockedSignals tests blocked signal handling.
func TestBlockedSignals(t *testing.T) {
	p := NewProcess(1, 0, "test", nil)
	p.SignalMask = make(SignalSet)
	p.SignalMask[SignalInterrupt] = true

	received := false
	p.SetSignalHandler(SignalInterrupt, func(proc *Process, sig Signal) {
		received = true
	})

	p.HandleSignal(SignalInterrupt)
	if received {
		t.Error("Signal handler should not be called for blocked signal")
	}
}

// TestSignalSetOperations tests signal set operations.
func TestSignalSetOperations(t *testing.T) {
	ss := NewSignalSet(SignalInterrupt, SignalKill)

	if ss.Len() != 2 {
		t.Errorf("Len() = %d, want 2", ss.Len())
	}

	// Difference
	diff := ss.Difference(NewSignalSet(SignalInterrupt))
	if !diff.Contains(SignalKill) {
		t.Error("Difference should still contain SignalKill")
	}
	if diff.Contains(SignalInterrupt) {
		t.Error("Difference should not contain SignalInterrupt")
	}
}
