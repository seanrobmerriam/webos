// wsh is the WebOS Shell - a POSIX-compliant command-line shell.
//
// Usage:
//
//	wsh [options] [command_file]
//
// Options:
//
//	-c command   Execute command and exit
//	-i           Interactive mode
//	-v           Verbose mode
//
// For more information, see https://github.com/webos/wsh
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Parse command line flags
	command := flag.String("c", "", "Execute command and exit")
	interactive := flag.Bool("i", false, "Force interactive mode")
	verbose := flag.Bool("v", false, "Verbose mode")
	flag.Parse()

	// Get command file from arguments
	args := flag.Args()
	var cmdFile string
	if len(args) > 0 {
		cmdFile = args[0]
	}

	// Create shell
	shell := NewShell()
	shell.Interactive = *interactive || (cmdFile == "" && *command == "")

	if *verbose {
		shell.SetPrompt("wsh> ")
	}

	// Execute single command
	if *command != "" {
		status, err := shell.ExecuteString(*command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wsh: %s\n", err)
			os.Exit(1)
		}
		os.Exit(status)
	}

	// Execute command file
	if cmdFile != "" {
		f, err := os.Open(cmdFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wsh: %s: %s\n", cmdFile, err)
			os.Exit(1)
		}
		defer f.Close()
		shell.Stdin = f
		shell.Interactive = false
	}

	// Run shell
	if err := shell.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wsh: %s\n", err)
		os.Exit(1)
	}
}

// init sets up package-level configuration.
func init() {
	// Suppress unused variable warning
	_ = strings.TrimSpace
}
