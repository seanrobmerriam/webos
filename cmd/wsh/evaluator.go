// Package wsh implements the WebOS shell (wsh).
// This file provides command evaluation functionality.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"webos/pkg/parser"
)

// EvalResult represents the result of command evaluation.
type EvalResult struct {
	Status int
	Job    *Job
	Output string
}

// Evaluator evaluates shell commands.
type Evaluator struct {
	Env   map[string]string
	Debug bool
}

// NewEvaluator creates a new Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		Env:   make(map[string]string),
		Debug: false,
	}
}

// SetEnv sets an environment variable.
func (e *Evaluator) SetEnv(key, value string) {
	e.Env[key] = value
}

// GetEnv gets an environment variable.
func (e *Evaluator) GetEnv(key string) string {
	if val, ok := e.Env[key]; ok {
		return val
	}
	return os.Getenv(key)
}

// Eval evaluates a parsed command list.
func (e *Evaluator) Eval(list *parser.ListNode) (*EvalResult, error) {
	return e.evalList(list)
}

// evalList evaluates a list of pipelines.
func (e *Evaluator) evalList(list *parser.ListNode) (*EvalResult, error) {
	for i, pipeline := range list.Elements {
		result, err := e.evalPipeline(pipeline)
		if err != nil {
			return result, err
		}

		// Check for short-circuit evaluation
		if i < len(list.Sep) {
			if list.Sep[i] == parser.TokenBackground {
				// Background job already handled in evalPipeline
				continue
			}
		}

		// Check exit status for && and ||
		if pipeline.And && result.Status != 0 {
			// Stop on first failure with &&
			return result, nil
		}
		if pipeline.Or && result.Status == 0 {
			// Stop on first success with ||
			return result, nil
		}
	}

	return &EvalResult{Status: 0}, nil
}

// evalPipeline evaluates a pipeline of commands.
func (e *Evaluator) evalPipeline(pipeline *parser.PipelineNode) (*EvalResult, error) {
	if len(pipeline.Commands) == 0 {
		return &EvalResult{Status: 0}, nil
	}

	if len(pipeline.Commands) == 1 {
		return e.evalCommand(pipeline.Commands[0])
	}

	// Create pipeline of commands
	cmds := make([]*exec.Cmd, len(pipeline.Commands))
	for i, cmd := range pipeline.Commands {
		cmds[i] = e.buildCommand(cmd)
	}

	pipe := NewPipeline(cmds)
	if err := pipe.SetupPipes(); err != nil {
		return nil, err
	}

	// Set up environment
	env := os.Environ()
	for k, v := range e.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	pipe.SetEnv(env)

	// Set working directory
	pipe.SetDir(e.Env["PWD"])

	// Run pipeline
	if err := pipe.Run(); err != nil {
		return &EvalResult{Status: 1}, nil
	}

	return &EvalResult{Status: 0}, nil
}

// evalCommand evaluates a single command.
func (e *Evaluator) evalCommand(cmd *parser.CommandNode) (*EvalResult, error) {
	if cmd == nil || cmd.Name == "" {
		return &EvalResult{Status: 0}, nil
	}

	// Check for built-in commands
	if builtin := GetBuiltin(cmd.Name); builtin != nil {
		return e.evalBuiltin(builtin, append([]string{cmd.Name}, cmd.Args...))
	}

	// Build and run external command
	execCmd := e.buildCommand(cmd)
	execCmd.Env = os.Environ()
	for k, v := range e.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := execCmd.Run(); err != nil {
		if _, ok := err.(*os.PathError); ok {
			return &EvalResult{Status: 127}, nil // Command not found
		}
		return &EvalResult{Status: 1}, nil
	}

	return &EvalResult{Status: 0}, nil
}

// buildCommand builds an exec.Cmd from a CommandNode.
func (e *Evaluator) buildCommand(cmd *parser.CommandNode) *exec.Cmd {
	// Expand arguments
	args := make([]string, 0, len(cmd.Args)+1)
	args = append(args, cmd.Name)
	for _, arg := range cmd.Args {
		expanded := e.expandVariable(arg)
		args = append(args, expanded)
	}

	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Dir = e.Env["PWD"]

	return execCmd
}

// evalBuiltin evaluates a built-in command.
func (e *Evaluator) evalBuiltin(builtin *BuiltinCommand, args []string) (*EvalResult, error) {
	status := builtin.Func(args)
	return &EvalResult{Status: status}, nil
}

// expandVariable expands environment variables in a string.
func (e *Evaluator) expandVariable(s string) string {
	result := make([]byte, 0, len(s))
	i := 0

	for i < len(s) {
		if s[i] == '$' && i+1 < len(s) {
			switch s[i+1] {
			case '$':
				result = append(result, '$')
				i += 2
			case '?':
				result = append(result, []byte(fmt.Sprintf("%d", 0))...) // TODO: use last status
				i += 2
			case '{':
				// ${VAR}
				end := strings.Index(s[i+2:], "}")
				if end == -1 {
					result = append(result, s[i:]...)
					return string(result)
				}
				varName := s[i+2 : i+2+end]
				if val := e.GetEnv(varName); val != "" {
					result = append(result, val...)
				}
				i += 2 + end + 1
			default:
				// $VAR
				start := i + 1
				for start < len(s) && (s[start] == '_' || (s[start] >= 'a' && s[start] <= 'z') ||
					(s[start] >= 'A' && s[start] <= 'Z') || (s[start] >= '0' && s[start] <= '9')) {
					start++
				}
				varName := s[i+1 : start]
				if val := e.GetEnv(varName); val != "" {
					result = append(result, val...)
				}
				i = start
			}
		} else {
			result = append(result, s[i])
			i++
		}
	}

	return string(result)
}

// ExpandVariables expands all variables in a slice of strings.
func (e *Evaluator) ExpandVariables(args []string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = e.expandVariable(arg)
	}
	return result
}
