// Package wsh implements the WebOS shell (wsh).
// This file provides job control functionality for managing background jobs.
package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// JobState represents the state of a job.
type JobState int

// Job states.
const (
	JobRunning JobState = iota
	JobStopped
	JobDone
	JobTerminated
)

// Job represents a shell job with one or more processes.
type Job struct {
	ID         int
	Name       string
	State      JobState
	ExitStatus int
	Processes  []*Process
	PGID       int  // Process group ID
	Notify     bool // Whether to notify when state changes
	StartTime  time.Time
}

// Process represents a single process in a job.
type Process struct {
	PID    int
	Cmd    *exec.Cmd
	Job    *Job
	Status syscall.WaitStatus
}

// String returns a string representation of the job.
func (j *Job) String() string {
	stateStr := "Running"
	switch j.State {
	case JobStopped:
		stateStr = "Stopped"
	case JobDone:
		stateStr = "Done"
	case JobTerminated:
		stateStr = "Terminated"
	}
	return fmt.Sprintf("[%d] %s\t%s", j.ID, stateStr, j.Name)
}

// IsRunning returns true if the job is running.
func (j *Job) IsRunning() bool {
	return j.State == JobRunning || j.State == JobStopped
}

// IsCompleted returns true if the job is done.
func (j *Job) IsCompleted() bool {
	return j.State == JobDone
}

// UpdateStatus updates the job status based on process statuses.
func (j *Job) UpdateStatus() {
	allDone := true
	anyStopped := false

	for _, p := range j.Processes {
		if p.Status.Exited() {
			j.ExitStatus = p.Status.ExitStatus()
		} else if p.Status.Stopped() {
			anyStopped = true
			allDone = false
		} else if p.Status.Continued() {
			allDone = false
		} else {
			allDone = false
		}
	}

	if allDone {
		j.State = JobDone
	} else if anyStopped {
		j.State = JobStopped
	} else {
		j.State = JobRunning
	}
}

// Signal sends a signal to all processes in the job.
func (j *Job) Signal(sig syscall.Signal) error {
	if j.PGID == 0 {
		return fmt.Errorf("job %d has no process group", j.ID)
	}
	return syscall.Kill(-j.PGID, sig)
}

// Terminate sends SIGTERM to all processes in the job.
func (j *Job) Terminate() error {
	return j.Signal(syscall.SIGTERM)
}

// Kill sends SIGKILL to all processes in the job.
func (j *Job) Kill() error {
	return j.Signal(syscall.SIGKILL)
}

// Wait waits for all processes in the job to complete.
func (j *Job) Wait() error {
	for _, p := range j.Processes {
		if p.Cmd != nil && p.Cmd.Process != nil {
			if _, err := p.Cmd.Process.Wait(); err != nil {
				return err
			}
		}
	}
	return nil
}

// JobTable manages the shell's jobs.
type JobTable struct {
	jobs    []*Job
	nextID  int
	current *Job // Current job (last foreground job)
}

// NewJobTable creates a new job table.
func NewJobTable() *JobTable {
	return &JobTable{
		jobs:   make([]*Job, 0),
		nextID: 1,
	}
}

// AddJob adds a new job to the table.
func (t *JobTable) AddJob(job *Job) {
	job.ID = t.nextID
	t.nextID++
	t.jobs = append(t.jobs, job)
	t.current = job
}

// RemoveJob removes a job from the table.
func (t *JobTable) RemoveJob(id int) {
	for i, j := range t.jobs {
		if j.ID == id {
			t.jobs = append(t.jobs[:i], t.jobs[i+1:]...)
			return
		}
	}
}

// GetJob returns a job by ID.
func (t *JobTable) GetJob(id int) *Job {
	for _, j := range t.jobs {
		if j.ID == id {
			return j
		}
	}
	return nil
}

// GetJobByPID returns a job containing the given PID.
func (t *JobTable) GetJobByPID(pid int) *Job {
	for _, j := range t.jobs {
		for _, p := range j.Processes {
			if p.PID == pid {
				return j
			}
		}
	}
	return nil
}

// GetCurrent returns the current job.
func (t *JobTable) GetCurrent() *Job {
	return t.current
}

// GetJobs returns all jobs.
func (t *JobTable) GetJobs() []*Job {
	return t.jobs
}

// JobsWithState returns all jobs with the given state.
func (t *JobTable) JobsWithState(state JobState) []*Job {
	result := make([]*Job, 0)
	for _, j := range t.jobs {
		if j.State == state {
			result = append(result, j)
		}
	}
	return result
}

// CleanupCompleted removes all completed jobs.
func (t *JobTable) CleanupCompleted() {
	active := make([]*Job, 0)
	for _, j := range t.jobs {
		if !j.IsCompleted() {
			active = append(active, j)
		}
	}
	t.jobs = active
}

// FormatJobID parses a job ID string and returns the job ID.
func FormatJobID(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty job ID")
	}

	// Handle %% and %+
	if s == "%%" || s == "%+" {
		return -1, nil // Current job
	}
	// Handle %-
	if s == "%-" {
		return -2, nil // Previous job
	}

	// Handle %n (by name)
	if len(s) > 1 && s[0] == '%' {
		name := s[1:]
		return 0, fmt.Errorf("job name: %s", name)
	}

	// Handle numeric ID
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid job ID: %s", s)
	}
	return id, nil
}
