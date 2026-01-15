package main

import (
	"fmt"
	"log"
	"time"

	"webos/pkg/process"
	"webos/pkg/process/ipc"
)

func main() {
	fmt.Println("=== WebOS Process Management Demo ===")
	fmt.Println()

	// Create scheduler
	scheduler := process.NewPriorityScheduler()
	fmt.Println("Created priority scheduler")

	// Create process manager
	pm := process.NewProcessManager(scheduler)
	fmt.Println("Created process manager")

	// Demonstrate process creation
	fmt.Println("\n--- Process Creation ---")
	config := &process.CreateConfig{
		Command:  "demo-process",
		Args:     []string{"--mode", "interactive"},
		Env:      []string{"PATH=/bin", "HOME=/root"},
		Cwd:      "/",
		Priority: process.PriorityNormal,
		Limits:   process.DefaultLimits(),
	}

	p1, err := pm.CreateProcess(config)
	if err != nil {
		log.Fatalf("Failed to create process: %v", err)
	}
	fmt.Printf("Created process: PID=%d, Command=%s\n", p1.PID, p1.Command)

	// Create another process with different priority
	config2 := &process.CreateConfig{
		Command:  "background-worker",
		Args:     []string{"--tasks", "100"},
		Priority: process.PriorityLow,
	}
	p2, err := pm.CreateProcess(config2)
	if err != nil {
		log.Fatalf("Failed to create process 2: %v", err)
	}
	fmt.Printf("Created background process: PID=%d, Priority=%v\n", p2.PID, p2.Priority)

	// Demonstrate state transitions
	fmt.Println("\n--- Process State Transitions ---")
	fmt.Printf("Initial state: %s\n", p1.GetState())

	err = pm.Start(p1.PID)
	if err != nil {
		log.Fatalf("Failed to start process: %v", err)
	}
	fmt.Printf("After Start: %s\n", p1.GetState())

	err = p1.Yield()
	if err != nil {
		log.Fatalf("Failed to yield: %v", err)
	}
	fmt.Printf("After Yield: %s\n", p1.GetState())

	// Demonstrate forking
	fmt.Println("\n--- Process Forking ---")
	parent, _ := pm.CreateProcess(&process.CreateConfig{Command: "parent"})
	child, err := pm.Fork(parent, &process.CreateConfig{Command: "child"})
	if err != nil {
		log.Fatalf("Failed to fork: %v", err)
	}
	fmt.Printf("Parent PID: %d, Child PID: %d\n", parent.PID, child.PID)
	fmt.Printf("Child CWD inherited: %s\n", child.Cwd)

	// Demonstrate IPC - Pipes
	fmt.Println("\n--- IPC: Anonymous Pipes ---")
	pipePair := ipc.NewPipePair()
	fmt.Println("Created pipe pair")

	// Demo pipe communication
	go func() {
		data := []byte("Hello from writer!")
		pipePair.Write.Write(data)
		fmt.Printf("Wrote %d bytes to pipe\n", len(data))
	}()

	buf := make([]byte, 1024)
	n, err := pipePair.Read.Read(buf)
	if err != nil {
		log.Fatalf("Failed to read from pipe: %v", err)
	}
	fmt.Printf("Read %d bytes from pipe: %s\n", n, string(buf[:n]))

	// Demonstrate IPC - Message Queue
	fmt.Println("\n--- IPC: Message Queues ---")
	msgQueue := ipc.NewMessageQueue()

	// Send messages
	msg1 := ipc.NewMessage(ipc.MessageTypeData, []byte("First message"), ipc.MessagePriorityNormal, 1)
	msg2 := ipc.NewMessage(ipc.MessageTypeData, []byte("High priority"), ipc.MessagePriorityHigh, 1)

	err = msgQueue.Send(msg1)
	if err != nil {
		log.Fatalf("Failed to send message 1: %v", err)
	}
	err = msgQueue.Send(msg2)
	if err != nil {
		log.Fatalf("Failed to send message 2: %v", err)
	}
	fmt.Println("Sent 2 messages (one high priority)")

	// Receive (should get high priority first)
	received, err := msgQueue.Receive()
	if err != nil {
		log.Fatalf("Failed to receive message: %v", err)
	}
	fmt.Printf("Received message: %s (priority: %d)\n", string(received.Payload), received.Priority)

	// Demonstrate IPC - Shared Memory
	fmt.Println("\n--- IPC: Shared Memory ---")
	shmManager := ipc.NewSharedMemoryManager()
	seg, err := shmManager.CreateSegment(1, "demo-segment", 1024)
	if err != nil {
		log.Fatalf("Failed to create shared memory: %v", err)
	}
	fmt.Printf("Created shared memory segment: ID=%s, Size=%d\n", seg.ID, seg.Size())

	// Write to shared memory
	testData := []byte("Shared memory test data!")
	_, err = seg.Write(0, testData)
	if err != nil {
		log.Fatalf("Failed to write to shared memory: %v", err)
	}
	fmt.Printf("Wrote %d bytes to shared memory\n", len(testData))

	// Read from shared memory
	readBuf := make([]byte, 1024)
	n, err = seg.Read(0, readBuf)
	if err != nil {
		log.Fatalf("Failed to read from shared memory: %v", err)
	}
	fmt.Printf("Read %d bytes from shared memory: %s\n", n, string(readBuf[:n]))

	// Demonstrate Signal Handling
	fmt.Println("\n--- Signal Handling ---")
	signalManager := ipc.NewSignalManager()

	receivedSignal := false
	signalManager.RegisterHandler(ipc.SignalInterrupt, func(pid int, sig ipc.Signal) {
		receivedSignal = true
		fmt.Printf("Received signal %d for process %d\n", sig, pid)
	})

	err = signalManager.Send(p1.PID, ipc.SignalInterrupt)
	if err != nil {
		log.Fatalf("Failed to send signal: %v", err)
	}
	fmt.Println("Sent SIGINT signal")

	// Process pending signals
	signalManager.ProcessSignals(p1.PID, func(pid int, sig ipc.Signal) {
		fmt.Printf("Handler processed signal %d for PID %d\n", sig, pid)
	})
	if receivedSignal {
		fmt.Println("Signal was successfully delivered to handler")
	}

	// Demonstrate Resource Limits
	fmt.Println("\n--- Resource Limits ---")
	enforcer := process.NewEnforcer()
	enforcer.SetLimits(p1.PID, &process.ResourceLimits{
		MaxMemory: 100 * 1024 * 1024, // 100 MB
		MaxFiles:  10,
		CPUTime:   time.Hour,
	})

	err = enforcer.AddFileUsage(p1.PID)
	if err != nil {
		log.Fatalf("Failed to add file usage: %v", err)
	}
	fmt.Printf("Process %d file usage: 1\n", p1.PID)

	err = enforcer.CheckLimits(p1.PID)
	if err != nil {
		log.Fatalf("Limit check failed: %v", err)
	}
	fmt.Println("Resource limits check passed")

	// Demonstrate scheduler statistics
	fmt.Println("\n--- Scheduler Statistics ---")
	stats := scheduler.Stats()
	fmt.Printf("Scheduler stats: Total Scheduled=%d, Current Priority=%d\n",
		stats.TotalScheduled, stats.CurrentPriority)

	lengths := stats.QueueLengths
	fmt.Printf("Queue lengths: Low=%d, Normal=%d, High=%d, Critical=%d\n",
		lengths[0], lengths[1], lengths[2], lengths[3])

	// Demonstrate process termination
	fmt.Println("\n--- Process Termination ---")
	// Start the process again before terminating
	err = pm.Start(p1.PID)
	if err != nil {
		log.Fatalf("Failed to start process: %v", err)
	}
	err = pm.Terminate(p1.PID, 0)
	if err != nil {
		log.Fatalf("Failed to terminate process: %v", err)
	}
	fmt.Printf("Process %d terminated with state: %s\n", p1.PID, p1.GetState())

	// Demonstrate process listing
	fmt.Println("\n--- Process Listing ---")
	processes := pm.GetProcesses()
	fmt.Printf("Total processes: %d\n", len(processes))
	for _, p := range processes {
		fmt.Printf("  PID=%d, Command=%s, State=%s\n", p.PID, p.Command, p.GetState())
	}

	fmt.Println("\n=== Demo Complete ===")
}
