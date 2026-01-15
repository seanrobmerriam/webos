/*
Package process provides process management functionality for the WebOS system.

This package implements a virtual process management system with cooperative
multitasking, inspired by the Unix process model. It includes:

  - Process lifecycle management (creation, termination, state transitions)
  - Priority-based task scheduling
  - Process state machine with multiple states (running, waiting, ready, zombie, stopped)
  - Resource limits and quotas
  - Inter-Process Communication (IPC) primitives:
  - Anonymous pipes
  - Named pipes (FIFO)
  - Message queues
  - Shared memory
  - Signal handling

# Process States

Processes can be in one of the following states:

  - Running: Process is currently executing
  - Waiting: Process is blocked waiting for I/O or other events
  - Ready: Process is ready to run but waiting for CPU time
  - Zombie: Process has terminated but parent hasn't collected its status
  - Stopped: Process has been stopped (e.g., by a signal)

# Usage

Creating a new process:

	p, err := manager.CreateProcess(&CreateConfig{
		Command:     "echo",
		Args:        []string{"hello world"},
		Env:         []string{"PATH=/bin"},
		Cwd:         "/",
		Priority:    PriorityNormal,
	})

	if err != nil {
		// Handle error
	}

	// Start the process
	err = manager.Start(p.PID)
	if err != nil {
		// Handle error
	}

# Resource Limits

Each process can have resource limits applied to control:

  - CPU time
  - Memory usage
  - Number of open files
  - Stack size
  - Other system resources

Example:

	limits := &ResourceLimits{
		CPUTime:      time.Minute,
		MaxMemory:    100 * 1024 * 1024, // 100 MB
		MaxFiles:     64,
		MaxStackSize: 8 * 1024 * 1024,   // 8 MB
	}

# Inter-Process Communication

The package provides various IPC mechanisms for communication between processes:

  - Pipes: Anonymous byte streams for parent-child communication
  - Named Pipes: Persistent FIFO files for unrelated process communication
  - Message Queues: Structured message passing with priorities
  - Shared Memory: Fast memory sharing with synchronization
  - Signals: Asynchronous notification mechanism
*/
package process
