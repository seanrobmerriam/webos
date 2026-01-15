package process

import (
	"container/heap"
	"sync"
	"time"
)

// Scheduler interface defines the contract for process scheduling.
type Scheduler interface {
	// Schedule adds a process to the run queue.
	Schedule(p *Process) error
	// Yield yields the CPU to the scheduler.
	Yield()
	// GetNextRunnable returns the next process to run.
	GetNextRunnable() *Process
	// Remove removes a process from the scheduler.
	Remove(pid int) error
	// Len returns the number of runnable processes.
	Len() int
}

// PriorityScheduler implements priority-based round-robin scheduling.
type PriorityScheduler struct {
	// queues holds run queues for each priority level.
	queues [4]*ProcessQueue
	// mu protects the scheduler state.
	mu sync.Mutex
	// currentPriority is the current priority being serviced.
	currentPriority int
	// quantum is the time slice for each process.
	quantum time.Duration
	// tickCount tracks scheduler ticks.
	tickCount int64
}

// ProcessQueue is a priority queue of processes.
type ProcessQueue struct {
	items []*Process
	index map[int]int
}

// NewProcessQueue creates a new process queue.
func NewProcessQueue() *ProcessQueue {
	return &ProcessQueue{
		items: make([]*Process, 0),
		index: make(map[int]int),
	}
}

// Len returns the number of items in the queue.
func (q *ProcessQueue) Len() int { return len(q.items) }

// Less implements heap.Interface - lower priority means higher in queue.
func (q *ProcessQueue) Less(i, j int) bool {
	return q.items[i].PID < q.items[j].PID
}

// Swap swaps two items in the queue.
func (q *ProcessQueue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
	q.index[q.items[i].PID] = i
	q.index[q.items[j].PID] = j
}

// Push adds an item to the queue.
func (q *ProcessQueue) Push(x interface{}) {
	p := x.(*Process)
	q.index[p.PID] = len(q.items)
	q.items = append(q.items, p)
}

// Pop removes and returns the first item from the queue.
func (q *ProcessQueue) Pop() interface{} {
	old := q.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	q.items = old[0 : n-1]
	delete(q.index, item.PID)
	return item
}

// Peek returns the first item without removing it.
func (q *ProcessQueue) Peek() *Process {
	if len(q.items) == 0 {
		return nil
	}
	return q.items[0]
}

// Contains checks if a process is in the queue.
func (q *ProcessQueue) Contains(pid int) bool {
	_, ok := q.index[pid]
	return ok
}

// Remove removes a process from the queue.
func (q *ProcessQueue) Remove(pid int) bool {
	if idx, ok := q.index[pid]; ok {
		q.Swap(idx, len(q.items)-1)
		q.items = q.items[:len(q.items)-1]
		delete(q.index, pid)
		return true
	}
	return false
}

// Round-robin queue implementation.
type RRQueue struct {
	items []*Process
	head  int
}

// NewRRQueue creates a new round-robin queue.
func NewRRQueue() *RRQueue {
	return &RRQueue{
		items: make([]*Process, 0),
		head:  0,
	}
}

// Len returns the number of items in the queue.
func (q *RRQueue) Len() int { return len(q.items) }

// Push adds an item to the queue.
func (q *RRQueue) Push(p *Process) {
	q.items = append(q.items, p)
}

// Pop removes and returns the next item (round-robin).
func (q *RRQueue) Pop() *Process {
	if q.Len() == 0 {
		return nil
	}
	p := q.items[q.head]
	q.head++
	if q.head >= q.Len() {
		q.head = 0
	}
	return p
}

// Contains checks if a process is in the queue.
func (q *RRQueue) Contains(pid int) bool {
	for _, p := range q.items {
		if p.PID == pid {
			return true
		}
	}
	return false
}

// Remove removes a process from the queue.
func (q *RRQueue) Remove(pid int) bool {
	for i, p := range q.items {
		if p.PID == pid {
			q.items = append(q.items[:i], q.items[i+1:]...)
			if q.head > i {
				q.head--
			}
			return true
		}
	}
	return false
}

// heap.Interface implementation for ProcessQueue.
var _ heap.Interface = (*ProcessQueue)(nil)

// NewPriorityScheduler creates a new priority scheduler.
func NewPriorityScheduler() *PriorityScheduler {
	ps := &PriorityScheduler{
		quantum:         100 * time.Millisecond,
		currentPriority: int(PriorityLow),
	}

	for i := range ps.queues {
		ps.queues[i] = NewProcessQueue()
	}

	heap.Init(ps.queues[0])

	return ps
}

// SetQuantum sets the time quantum for each process.
func (s *PriorityScheduler) SetQuantum(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quantum = d
}

// GetQuantum returns the current time quantum.
func (s *PriorityScheduler) GetQuantum() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quantum
}

// Schedule adds a process to the run queue.
func (s *PriorityScheduler) Schedule(p *Process) error {
	if p == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Only schedule runnable processes
	if !p.IsRunnable() {
		return nil
	}

	// Add to appropriate priority queue
	heap.Push(s.queues[p.Priority], p)

	return nil
}

// Yield yields the CPU to the scheduler.
func (s *PriorityScheduler) Yield() {
	// In a real implementation, this would trigger context switch
	s.tickCount++
}

// GetNextRunnable returns the next process to run.
func (s *PriorityScheduler) GetNextRunnable() *Process {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check priority levels from highest to lowest (PriorityCritical to PriorityLow)
	// PriorityCritical=3, PriorityHigh=2, PriorityNormal=1, PriorityLow=0
	for i := 3; i >= 0; i-- {
		if s.queues[i].Len() > 0 {
			s.currentPriority = i
			return heap.Pop(s.queues[i]).(*Process)
		}
	}

	return nil
}

// Len returns the number of runnable processes.
func (s *PriorityScheduler) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := 0
	for _, q := range s.queues {
		total += q.Len()
	}
	return total
}

// Remove removes a process from the scheduler.
func (s *PriorityScheduler) Remove(pid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, q := range s.queues {
		if q.Remove(pid) {
			return nil
		}
	}

	return nil
}

// GetQueueLength returns the number of processes in each priority queue.
func (s *PriorityScheduler) GetQueueLengths() [4]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lengths [4]int
	for i, q := range s.queues {
		lengths[i] = q.Len()
	}
	return lengths
}

// Stats returns scheduler statistics.
func (s *PriorityScheduler) Stats() SchedulerStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lengths [4]int
	for i, q := range s.queues {
		lengths[i] = q.Len()
	}

	return SchedulerStats{
		TotalScheduled:  s.tickCount,
		QueueLengths:    lengths,
		CurrentPriority: s.currentPriority,
	}
}

// SchedulerStats contains scheduler statistics.
type SchedulerStats struct {
	TotalScheduled  int64
	QueueLengths    [4]int
	CurrentPriority int
}

// CooperativeScheduler implements cooperative multitasking.
type CooperativeScheduler struct {
	*PriorityScheduler
	// runningProcess is the currently running process.
	runningProcess *Process
	// waiting is a map of processes waiting for events.
	waiting map[int]*Process
	// mu protects running state.
	runMu sync.Mutex
}

// NewCooperativeScheduler creates a new cooperative scheduler.
func NewCooperativeScheduler() *CooperativeScheduler {
	return &CooperativeScheduler{
		PriorityScheduler: NewPriorityScheduler(),
		waiting:           make(map[int]*Process),
	}
}

// Run starts the scheduler loop.
func (s *CooperativeScheduler) Run() {
	for {
		p := s.GetNextRunnable()
		if p == nil {
			break
		}

		s.runMu.Lock()
		s.runningProcess = p
		s.runMu.Unlock()

		// Simulate running for the quantum
		p.AddCPUUsage(s.quantum)

		// Yield back to scheduler
		s.Yield()
	}
}

// GetRunningProcess returns the currently running process.
func (s *CooperativeScheduler) GetRunningProcess() *Process {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return s.runningProcess
}

// Block blocks a process until it's woken.
func (s *CooperativeScheduler) Block(pid int) error {
	p, err := GetProcessManager().GetProcess(pid)
	if err != nil {
		return err
	}

	if err := p.Wait(); err != nil {
		return err
	}

	s.mu.Lock()
	s.waiting[pid] = p
	s.mu.Unlock()

	return nil
}

// Wake wakes a blocked process.
func (s *CooperativeScheduler) Wake(pid int) error {
	s.mu.Lock()
	p, ok := s.waiting[pid]
	delete(s.waiting, pid)
	s.mu.Unlock()

	if !ok {
		return ErrProcessNotFound
	}

	return p.Wake()
}

// processManager holder for global access.
var processManager *ProcessManager

// SetProcessManager sets the global process manager.
func SetProcessManager(pm *ProcessManager) {
	processManager = pm
}

// GetProcessManager gets the global process manager.
func GetProcessManager() *ProcessManager {
	return processManager
}
