// Package wsh implements the WebOS shell (wsh).
// This file provides tests for the shell functionality.
package main

import (
	"os"
	"strings"
	"testing"
)

func TestBuiltinCommands(t *testing.T) {
	tests := []struct {
		name     string
		builtin  string
		args     []string
		expected int
	}{
		{"echo basic", "echo", []string{"echo", "hello"}, 0},
		{"echo newline", "echo", []string{"echo", "-n", "hello"}, 0},
		{"pwd", "pwd", []string{"pwd"}, 0},
		{"cd", "cd", []string{"cd", "/"}, 0},
		{"cd invalid", "cd", []string{"cd", "/nonexistent/path"}, 1},
		{"exit 0", "exit", []string{"exit", "0"}, 0}, // Note: exits immediately
		{"help", "help", []string{"help"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip exit test as it actually exits
			if tt.builtin == "exit" {
				t.Skip("exit builtin terminates the program")
			}

			builtin := GetBuiltin(tt.builtin)
			if builtin == nil {
				t.Fatalf("expected builtin %s to exist", tt.builtin)
			}

			status := builtin.Func(tt.args)
			if status != tt.expected {
				t.Errorf("expected status %d, got %d", tt.expected, status)
			}
		})
	}
}

func TestIsBuiltin(t *testing.T) {
	builtins := []string{"cd", "pwd", "echo", "export", "set", "alias", "unalias", "history", "jobs", "fg", "bg", "exit", "help"}

	for _, name := range builtins {
		if !IsBuiltin(name) {
			t.Errorf("expected %s to be a builtin", name)
		}
	}

	nonBuiltins := []string{"ls", "cat", "grep", "make"}
	for _, name := range nonBuiltins {
		if IsBuiltin(name) {
			t.Errorf("expected %s NOT to be a builtin", name)
		}
	}
}

func TestJobTable(t *testing.T) {
	table := NewJobTable()

	// Add a job
	job := &Job{
		Name:      "test job",
		State:     JobRunning,
		Processes: []*Process{},
	}
	table.AddJob(job)

	if job.ID != 1 {
		t.Errorf("expected job ID 1, got %d", job.ID)
	}

	// Get job
	retrieved := table.GetJob(1)
	if retrieved == nil {
		t.Fatal("expected to get job 1")
	}
	if retrieved.Name != job.Name {
		t.Errorf("expected job name %s, got %s", job.Name, retrieved.Name)
	}

	// Get current job
	current := table.GetCurrent()
	if current == nil || current.ID != job.ID {
		t.Error("expected current job to be the added job")
	}

	// Get all jobs
	jobs := table.GetJobs()
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}

	// Remove job
	table.RemoveJob(1)
	if table.GetJob(1) != nil {
		t.Error("expected job to be removed")
	}
}

func TestJobStates(t *testing.T) {
	tests := []struct {
		state    JobState
		expected string
	}{
		{JobRunning, "Running"},
		{JobStopped, "Stopped"},
		{JobDone, "Done"},
		{JobTerminated, "Terminated"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			job := &Job{State: tt.state}
			s := job.String()
			if !strings.Contains(s, tt.expected) {
				t.Errorf("expected job string to contain %s, got %s", tt.expected, s)
			}
		})
	}
}

func TestJobIsRunning(t *testing.T) {
	tests := []struct {
		state    JobState
		expected bool
	}{
		{JobRunning, true},
		{JobStopped, true},
		{JobDone, false},
		{JobTerminated, false},
	}

	for _, tt := range tests {
		job := &Job{State: tt.state}
		if job.IsRunning() != tt.expected {
			t.Errorf("IsRunning() for state %d: expected %v, got %v", tt.state, tt.expected, job.IsRunning())
		}
	}
}

func TestShellExecuteString(t *testing.T) {
	shell := NewShell()
	shell.Interactive = false

	// Test basic echo
	status, err := shell.ExecuteString("echo hello")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if status != 0 {
		t.Errorf("expected status 0, got %d", status)
	}
}

func TestShellHistory(t *testing.T) {
	shell := NewShell()

	// Add to history
	shell.AddToHistory("cmd1")
	shell.AddToHistory("cmd2")

	history := shell.GetHistory()
	if len(history) != 2 {
		t.Errorf("expected 2 history items, got %d", len(history))
	}
	if history[0] != "cmd1" || history[1] != "cmd2" {
		t.Error("history items don't match expected values")
	}
}

func TestEvaluatorVariableExpansion(t *testing.T) {
	eval := NewEvaluator()

	// Use variables that are guaranteed to exist
	tests := []struct {
		input    string
		expected string
	}{
		{"$HOME", os.Getenv("HOME")},
		{"${HOME}", os.Getenv("HOME")},
		{"prefix_$HOME", "prefix_" + os.Getenv("HOME")},
		{"$", "$"},
		{"no_expansion", "no_expansion"},
		{"${USER}", os.Getenv("USER")},
	}

	for _, tt := range tests {
		result := eval.expandVariable(tt.input)
		if result != tt.expected {
			t.Errorf("expandVariable(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestEvaluatorSetEnv(t *testing.T) {
	eval := NewEvaluator()

	eval.SetEnv("FOO", "bar")
	if eval.GetEnv("FOO") != "bar" {
		t.Error("GetEnv returned wrong value after SetEnv")
	}

	// Test that GetEnv falls back to os.Getenv
	origHome := "test_home"
	os.Setenv("HOME", origHome)
	defer os.Setenv("HOME", "") // Restore

	if eval.GetEnv("HOME") != origHome {
		t.Error("GetEnv should fall back to os.Getenv")
	}
}

func TestNewShell(t *testing.T) {
	shell := NewShell()

	if shell.Eval == nil {
		t.Error("expected Evaluator to be initialized")
	}
	if shell.JobTable == nil {
		t.Error("expected JobTable to be initialized")
	}
	if shell.History == nil {
		t.Error("expected History to be initialized")
	}
	if shell.Prompt == "" {
		t.Error("expected Prompt to be set")
	}
}

func TestBuiltinAlias(t *testing.T) {
	// This is a simple test to ensure alias functionality works
	state := newShellState()

	// Set an alias
	state.Aliases["ll"] = "ls -l"

	// Check it was set
	if state.Aliases["ll"] != "ls -l" {
		t.Error("alias was not set correctly")
	}
}

func TestBuiltinExport(t *testing.T) {
	// Test export prints all env vars when no args
	state := newShellState()
	state.Env["TEST_KEY"] = "TEST_VALUE"

	// Just verify it doesn't crash
	builtinExport([]string{"export"})
}

func TestPipelineNewPipeline(t *testing.T) {
	// Create a simple pipeline
	pipe := NewPipeline(nil)
	if pipe == nil {
		t.Error("expected non-nil pipeline")
	}
}

func TestFormatJobID(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasErr   bool
	}{
		{"1", 1, false},
		{"123", 123, false},
		{"0", 0, false},
		{"%%", -1, false}, // Current job
		{"%+", -1, false}, // Current job
		{"%-", -2, false}, // Previous job
		{"", 0, true},     // Empty
		{"abc", 0, true},  // Invalid
	}

	for _, tt := range tests {
		id, err := FormatJobID(tt.input)
		if tt.hasErr && err == nil {
			t.Errorf("expected error for %q", tt.input)
		}
		if !tt.hasErr && err != nil {
			t.Errorf("unexpected error for %q: %s", tt.input, err)
		}
		if !tt.hasErr && id != tt.expected {
			t.Errorf("FormatJobID(%q): expected %d, got %d", tt.input, tt.expected, id)
		}
	}
}
