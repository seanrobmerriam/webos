package process

import (
	"errors"
	"time"
)

// State transition errors.
var (
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrProcessNotFound   = errors.New("process not found")
	ErrProcessRunning    = errors.New("process is still running")
)

// StateTransition represents a valid state transition.
type StateTransition struct {
	From ProcessState
	To   ProcessState
}

// ValidTransitions defines all valid state transitions.
var ValidTransitions = []StateTransition{
	// Start process: Ready -> Running
	{From: StateReady, To: StateRunning},
	// Block for I/O: Running -> Waiting
	{From: StateRunning, To: StateWaiting},
	// I/O complete: Waiting -> Ready
	{From: StateWaiting, To: StateReady},
	// Yield CPU: Running -> Ready
	{From: StateRunning, To: StateReady},
	// Normal exit: Running -> Zombie
	{From: StateRunning, To: StateZombie},
	// Exit while waiting: Waiting -> Zombie
	{From: StateWaiting, To: StateZombie},
	// Stop signal: Running -> Stopped
	{From: StateRunning, To: StateStopped},
	// Stop signal: Waiting -> Stopped
	{From: StateWaiting, To: StateStopped},
	// Continue signal: Stopped -> Ready
	{From: StateStopped, To: StateReady},
	// Continue signal: Stopped -> Running
	{From: StateStopped, To: StateRunning},
}

// IsValidTransition checks if a state transition is valid.
func IsValidTransition(from, to ProcessState) bool {
	for _, t := range ValidTransitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

// CanTransition checks if a process can transition to the given state.
func (p *Process) CanTransition(to ProcessState) bool {
	return IsValidTransition(p.State, to)
}

// TransitionTo attempts to transition the process to a new state.
func (p *Process) TransitionTo(to ProcessState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !IsValidTransition(p.State, to) {
		return ErrInvalidTransition
	}

	p.State = to

	// Update timestamps on state changes
	switch to {
	case StateRunning:
		if p.StartedAt.IsZero() {
			p.StartedAt = time.Now()
		}
	case StateZombie:
		p.FinishedAt = time.Now()
	}

	return nil
}

// Start transitions a process from Ready to Running.
func (p *Process) Start() error {
	return p.TransitionTo(StateRunning)
}

// Stop transitions a process to the Stopped state.
func (p *Process) Stop() error {
	return p.TransitionTo(StateStopped)
}

// Wait blocks the process (transition to Waiting state).
func (p *Process) Wait() error {
	return p.TransitionTo(StateWaiting)
}

// Wake transitions a process from Waiting to Ready.
func (p *Process) Wake() error {
	return p.TransitionTo(StateReady)
}

// Yield transitions a running process to Ready (yielding CPU).
func (p *Process) Yield() error {
	return p.TransitionTo(StateReady)
}

// Terminate transitions a process to Zombie state.
func (p *Process) Terminate(exitCode int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !IsValidTransition(p.State, StateZombie) {
		return ErrInvalidTransition
	}

	p.State = StateZombie
	p.ExitCode = exitCode
	p.FinishedAt = time.Now()

	return nil
}

// Continue transitions a stopped process to Ready.
func (p *Process) Continue() error {
	return p.TransitionTo(StateReady)
}

// IsAlive returns true if the process is still active.
func (p *Process) IsAlive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.State != StateZombie && p.State != StateStopped
}

// IsTerminated returns true if the process has terminated.
func (p *Process) IsTerminated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.State == StateZombie
}

// IsStopped returns true if the process is stopped.
func (p *Process) IsStopped() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.State == StateStopped
}

// StateDuration returns how long the process has been in its current state.
func (p *Process) StateDuration() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	var startTime time.Time
	switch p.State {
	case StateReady:
		startTime = p.CreatedAt
	case StateRunning:
		startTime = p.StartedAt
	case StateWaiting:
		startTime = p.StartedAt
	case StateStopped:
		startTime = p.StartedAt
	case StateZombie:
		startTime = p.FinishedAt
	}

	return time.Since(startTime)
}

// TotalLifetime returns the total time since process creation.
func (p *Process) TotalLifetime() time.Duration {
	return time.Since(p.CreatedAt)
}

// ExecutionTime returns the total CPU time used.
func (p *Process) ExecutionTime() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State == StateZombie {
		return p.FinishedAt.Sub(p.StartedAt)
	}

	return time.Since(p.StartedAt)
}
