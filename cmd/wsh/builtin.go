// Package wsh implements the WebOS shell (wsh).
// This file provides built-in commands for the shell.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// BuiltinFunc is a function type for built-in commands.
type BuiltinFunc func(args []string) int

// BuiltinCommand represents a built-in command.
type BuiltinCommand struct {
	Name string
	Func BuiltinFunc
	Help string
}

// builtins holds all built-in commands.
var builtins = []BuiltinCommand{
	{"cd", builtinCd, "Change the current directory"},
	{"pwd", builtinPwd, "Print the current working directory"},
	{"echo", builtinEcho, "Display a line of text"},
	{"export", builtinExport, "Set environment variables"},
	{"set", builtinSet, "Set shell options"},
	{"alias", builtinAlias, "Create an alias"},
	{"unalias", builtinUnalias, "Remove an alias"},
	{"history", builtinHistory, "Display command history"},
	{"jobs", builtinJobs, "List background jobs"},
	{"fg", builtinFg, "Bring a job to the foreground"},
	{"bg", builtinBg, "Resume a job in the background"},
	{"exit", builtinExit, "Exit the shell"},
	{"help", builtinHelp, "Show this help message"},
}

// builtinMap maps command names to built-in commands.
var builtinMap = make(map[string]*BuiltinCommand)

// shellState holds the shell's state.
type shellState struct {
	Env        map[string]string
	Aliases    map[string]string
	History    []string
	JobTable   *JobTable
	WorkingDir string
}

// newShellState creates a new shell state.
func newShellState() *shellState {
	wd, _ := os.Getwd()
	return &shellState{
		Env:        make(map[string]string),
		Aliases:    make(map[string]string),
		History:    make([]string, 0),
		JobTable:   NewJobTable(),
		WorkingDir: wd,
	}
}

func init() {
	for i := range builtins {
		builtinMap[builtins[i].Name] = &builtins[i]
	}
}

// GetBuiltin returns the built-in command with the given name.
func GetBuiltin(name string) *BuiltinCommand {
	return builtinMap[name]
}

// IsBuiltin returns true if the command is a built-in.
func IsBuiltin(name string) bool {
	_, ok := builtinMap[name]
	return ok
}

// Built-in command implementations

// builtinCd changes the current directory.
func builtinCd(args []string) int {
	dir := os.Getenv("HOME")
	if len(args) > 1 {
		dir = args[1]
	} else if len(args) == 0 {
		// Already set to HOME above
	} else if args[0] == "-" {
		// cd to previous directory
		dir = os.Getenv("OLDPWD")
		if dir == "" {
			fmt.Fprintln(os.Stderr, "cd: OLDPWD not set")
			return 1
		}
	}

	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: %s\n", dir, err)
		return 1
	}

	// Update OLDPWD
	os.Setenv("OLDPWD", dir)
	// Update PWD handled by OS
	return 0
}

// builtinPwd prints the current working directory.
func builtinPwd(args []string) int {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pwd: %s\n", err)
		return 1
	}
	fmt.Println(wd)
	return 0
}

// builtinEcho prints its arguments.
func builtinEcho(args []string) int {
	// Handle -n flag
	n := false
	start := 1
	for i := 1; i < len(args); i++ {
		if args[i] == "-n" {
			n = true
		} else {
			start = i
			break
		}
	}

	if start < len(args) {
		fmt.Print(strings.Join(args[start:], " "))
	}
	if !n {
		fmt.Println()
	}
	return 0
}

// builtinExport sets environment variables.
func builtinExport(args []string) int {
	if len(args) == 1 {
		// Print all environment variables
		for _, env := range os.Environ() {
			fmt.Println(env)
		}
		return 0
	}

	for i := 1; i < len(args); i++ {
		parts := strings.SplitN(args[i], "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "export: %s: not a valid assignment\n", args[i])
			return 1
		}
		os.Setenv(parts[0], parts[1])
	}
	return 0
}

// builtinSet sets shell options.
func builtinSet(args []string) int {
	if len(args) == 1 {
		// Print all shell variables
		fmt.Println("set: shell options (not fully implemented)")
		return 0
	}

	// Handle options
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			// Set option
			fmt.Printf("set: %s (not fully implemented)\n", arg)
		} else if strings.HasPrefix(arg, "+") {
			// Unset option
			fmt.Printf("set: %s (not fully implemented)\n", arg)
		}
	}
	return 0
}

// builtinAlias creates an alias.
func builtinAlias(args []string) int {
	state := getShellState()

	if len(args) == 1 {
		// Print all aliases
		for name, value := range state.Aliases {
			fmt.Printf("alias %s='%s'\n", name, value)
		}
		return 0
	}

	for i := 1; i < len(args); i++ {
		parts := strings.SplitN(args[i], "=", 2)
		if len(parts) == 2 {
			state.Aliases[parts[0]] = parts[1]
		} else {
			// Print single alias
			if alias, ok := state.Aliases[args[i]]; ok {
				fmt.Printf("alias %s='%s'\n", args[i], alias)
			} else {
				fmt.Printf("alias: %s: not found\n", args[i])
			}
		}
	}
	return 0
}

// builtinUnalias removes an alias.
func builtinUnalias(args []string) int {
	state := getShellState()

	if len(args) == 1 {
		fmt.Fprintln(os.Stderr, "unalias: missing operand")
		return 1
	}

	for i := 1; i < len(args); i++ {
		delete(state.Aliases, args[i])
	}
	return 0
}

// builtinHistory displays command history.
func builtinHistory(args []string) int {
	state := getShellState()

	// Get history file path
	home := os.Getenv("HOME")
	histFile := home + "/.wsh_history"

	// Check for -c flag (clear history)
	if len(args) > 1 && args[1] == "-c" {
		state.History = make([]string, 0)
		os.Remove(histFile)
		return 0
	}

	// Try to read history file if not loaded
	if len(state.History) == 0 {
		if f, err := os.Open(histFile); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				state.History = append(state.History, scanner.Text())
			}
		}
	}

	// Print history
	start := 1
	if len(args) > 1 && args[1][0] == '-' {
		// Parse number
		var n int
		fmt.Sscanf(args[1], "%d", &n)
		if n > 0 && n < len(state.History) {
			start = len(state.History) - n
		}
	}

	for i := start; i < len(state.History); i++ {
		fmt.Printf("%d  %s\n", i+1, state.History[i])
	}
	return 0
}

// builtinJobs lists background jobs.
func builtinJobs(args []string) int {
	state := getShellState()
	jobs := state.JobTable.GetJobs()

	if len(jobs) == 0 {
		return 0
	}

	for _, job := range jobs {
		fmt.Println(job)
	}
	return 0
}

// builtinFg brings a job to the foreground.
func builtinFg(args []string) int {
	state := getShellState()

	if len(args) == 1 {
		fmt.Fprintln(os.Stderr, "fg: missing job ID")
		return 1
	}

	job, err := parseJobArg(state, args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "fg: %s\n", err)
		return 1
	}

	if job == nil {
		fmt.Fprintln(os.Stderr, "fg: no current job")
		return 1
	}

	// Signal job to continue in foreground
	if err := job.Signal(syscall.SIGCONT); err != nil {
		fmt.Fprintf(os.Stderr, "fg: %s\n", err)
		return 1
	}

	job.State = JobRunning
	state.JobTable.current = job
	return 0
}

// builtinBg resumes a job in the background.
func builtinBg(args []string) int {
	state := getShellState()

	if len(args) == 1 {
		fmt.Fprintln(os.Stderr, "bg: missing job ID")
		return 1
	}

	job, err := parseJobArg(state, args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "bg: %s\n", err)
		return 1
	}

	if job == nil {
		fmt.Fprintln(os.Stderr, "bg: no current job")
		return 1
	}

	// Signal job to continue in background
	if err := job.Signal(syscall.SIGCONT); err != nil {
		fmt.Fprintf(os.Stderr, "bg: %s\n", err)
		return 1
	}

	job.State = JobRunning
	return 0
}

// builtinExit exits the shell.
func builtinExit(args []string) int {
	code := 0
	if len(args) > 1 {
		fmt.Sscanf(args[1], "%d", &code)
	}
	os.Exit(code)
	return 0 // Never reached
}

// builtinHelp shows help information.
func builtinHelp(args []string) int {
	fmt.Println("WebOS Shell (wsh) - Built-in commands:")
	fmt.Println()
	for _, b := range builtinMap {
		fmt.Printf("  %-10s %s\n", b.Name, b.Help)
	}
	return 0
}

// Helper functions

func getShellState() *shellState {
	// In a real implementation, this would get the shell state from context
	return newShellState()
}

func parseJobArg(state *shellState, arg string) (*Job, error) {
	if arg == "%%" || arg == "%+" {
		return state.JobTable.GetCurrent(), nil
	}
	if arg == "%-" {
		// Return previous job (simplified)
		return nil, fmt.Errorf("no previous job")
	}

	id, err := fmt.Sscanf(arg, "%d")
	if err == nil && id > 0 {
		return state.JobTable.GetJob(id), nil
	}

	return nil, fmt.Errorf("invalid job specification: %s", arg)
}

// AddToHistory adds a command to the history.
func AddToHistory(cmd string) {
	state := getShellState()
	state.History = append(state.History, cmd)

	// Save to history file
	home := os.Getenv("HOME")
	histFile := home + "/.wsh_history"
	if f, err := os.OpenFile(histFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.WriteString(cmd + "\n")
		f.Close()
	}
}

// LookupAlias looks up an alias.
func LookupAlias(cmd string) string {
	state := getShellState()
	if alias, ok := state.Aliases[cmd]; ok {
		return alias
	}
	return cmd
}
