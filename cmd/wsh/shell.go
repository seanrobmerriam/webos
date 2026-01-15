// Package wsh implements the WebOS shell (wsh).
// This file provides the main shell loop and interface.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"webos/pkg/parser"
)

// Shell represents the interactive shell.
type Shell struct {
	Prompt      string
	Eval        *Evaluator
	JobTable    *JobTable
	History     []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool
}

// NewShell creates a new Shell instance.
func NewShell() *Shell {
	return &Shell{
		Prompt:      "$ ",
		Eval:        NewEvaluator(),
		JobTable:    NewJobTable(),
		History:     make([]string, 0),
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Interactive: isInteractive(),
	}
}

// isInteractive checks if stdin is a terminal.
func isInteractive() bool {
	return isTerminal(os.Stdin.Fd())
}

// isTerminal checks if the given file descriptor is a terminal.
func isTerminal(fd uintptr) bool {
	_, err := os.Stat("/dev/tty")
	return err == nil
}

// Run starts the shell and runs the main loop.
func (s *Shell) Run() error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTSTP)

	// Handle non-interactive mode
	if !s.Interactive {
		return s.runNonInteractive()
	}

	// Interactive mode
	scanner := bufio.NewScanner(s.Stdin)
	for {
		// Print prompt
		fmt.Fprint(s.Stdout, s.Prompt)

		// Check for signals
		select {
		case sig := <-sigChan:
			if sig == syscall.SIGINT {
				fmt.Fprintln(s.Stdout)
				continue
			}
		default:
		}

		// Read line
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			break // EOF
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Add to history
		s.History = append(s.History, line)
		AddToHistory(line)

		// Parse and evaluate
		if err := s.execute(line); err != nil {
			fmt.Fprintf(s.Stderr, "error: %s\n", err)
		}
	}

	return nil
}

// runNonInteractive executes commands from stdin.
func (s *Shell) runNonInteractive() error {
	scanner := bufio.NewScanner(s.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := s.execute(line); err != nil {
			fmt.Fprintf(s.Stderr, "error: %s\n", err)
			return err
		}
	}
	return nil
}

// execute parses and executes a command line.
func (s *Shell) execute(line string) error {
	// Check for alias expansion
	if alias := LookupAlias(strings.Fields(line)[0]); alias != "" {
		line = alias + " " + strings.TrimPrefix(line, strings.Fields(line)[0])
	}

	// Parse the command
	p := parser.NewParser(line)
	list, err := p.Parse()
	if err != nil {
		return err
	}

	// Evaluate the parsed command
	result, err := s.Eval.Eval(list)
	if err != nil {
		return err
	}

	// Update last exit status
	os.Setenv("?", fmt.Sprintf("%d", result.Status))

	return nil
}

// ExecuteString executes a command string and returns the result.
func (s *Shell) ExecuteString(cmd string) (int, error) {
	if err := s.execute(cmd); err != nil {
		return 1, err
	}
	return 0, nil
}

// SetPrompt sets the shell prompt.
func (s *Shell) SetPrompt(prompt string) {
	s.Prompt = prompt
}

// AddToHistory adds a command to the history.
func (s *Shell) AddToHistory(cmd string) {
	s.History = append(s.History, cmd)
}

// GetHistory returns the command history.
func (s *Shell) GetHistory() []string {
	return s.History
}
