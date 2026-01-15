// Package wsh implements the WebOS shell (wsh).
// This file provides pipeline handling for connecting commands.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Pipeline represents a pipeline of commands connected by pipes.
type Pipeline struct {
	Commands []*exec.Cmd
	Stdin    *os.File
	Stdout   *os.File
	Stderr   *os.File
}

// NewPipeline creates a new pipeline from commands.
func NewPipeline(cmds []*exec.Cmd) *Pipeline {
	return &Pipeline{
		Commands: cmds,
	}
}

// SetupPipes creates pipes between commands and sets up file descriptors.
func (p *Pipeline) SetupPipes() error {
	if len(p.Commands) == 0 {
		return nil
	}

	if len(p.Commands) == 1 {
		// Single command, use provided stdin/stdout
		return nil
	}

	// Create pipes for each connection
	for i := 0; i < len(p.Commands)-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("failed to create pipe: %w", err)
		}

		p.Commands[i].Stdout = w
		p.Commands[i+1].Stdin = r
	}

	return nil
}

// CleanupPipes closes all pipe file descriptors that were set up.
func (p *Pipeline) CleanupPipes() {
	// Close stdin/stdout of each command if they are pipes
	for _, cmd := range p.Commands {
		if cmd.Stdin != nil && cmd.Stdin != os.Stdin {
			if f, ok := cmd.Stdin.(*os.File); ok {
				f.Close()
			}
		}
		if cmd.Stdout != nil && cmd.Stdout != os.Stdout {
			if f, ok := cmd.Stdout.(*os.File); ok {
				f.Close()
			}
		}
		if cmd.Stderr != nil && cmd.Stderr != os.Stderr {
			if f, ok := cmd.Stderr.(*os.File); ok {
				f.Close()
			}
		}
	}
}

// Run executes the pipeline and waits for all commands to complete.
func (p *Pipeline) Run() error {
	// Set up process groups
	for i, cmd := range p.Commands {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		if i == 0 {
			// First command is the process group leader
			cmd.SysProcAttr.Setpgid = true
		}
	}

	// Start all commands
	for _, cmd := range p.Commands {
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start command: %w", err)
		}
	}

	// Close pipe write ends after starting
	for i := 0; i < len(p.Commands)-1; i++ {
		if cmdStdout, ok := p.Commands[i].Stdout.(*os.File); ok {
			cmdStdout.Close()
		}
		if cmdStdin, ok := p.Commands[i+1].Stdin.(*os.File); ok {
			cmdStdin.Close()
		}
	}

	// Wait for all commands
	var lastErr error
	for _, cmd := range p.Commands {
		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*os.PathError); ok {
				continue
			}
			lastErr = err
		}
	}

	return lastErr
}

// SetStdin sets the stdin for the first command.
func (p *Pipeline) SetStdin(f *os.File) {
	if len(p.Commands) > 0 {
		p.Commands[0].Stdin = f
	}
}

// SetStdout sets the stdout for the last command.
func (p *Pipeline) SetStdout(f *os.File) {
	if len(p.Commands) > 0 {
		p.Commands[len(p.Commands)-1].Stdout = f
	}
}

// SetStderr sets the stderr for the last command.
func (p *Pipeline) SetStderr(f *os.File) {
	if len(p.Commands) > 0 {
		p.Commands[len(p.Commands)-1].Stderr = f
	}
}

// SetEnv sets environment variables for all commands.
func (p *Pipeline) SetEnv(env []string) {
	for _, cmd := range p.Commands {
		cmd.Env = append(os.Environ(), env...)
	}
}

// SetDir sets the working directory for all commands.
func (p *Pipeline) SetDir(dir string) {
	for _, cmd := range p.Commands {
		cmd.Dir = dir
	}
}
