// Package main provides a demonstration program for the core utilities.
package main

import (
	"fmt"
	"os"

	"webos/cmd/utils/system"
)

// Demo demonstrates the core utilities.
func main() {
	fmt.Println("=== WebOS Core Utilities Demo ===")
	fmt.Println()

	// Demo system information (these work standalone)
	fmt.Println("--- System Information ---")
	demoSystemInfo()
	fmt.Println()

	// Demo text utilities (these work with io.Reader/io.Writer)
	fmt.Println("--- Text Processing ---")
	demoTextProcessing()
	fmt.Println()

	fmt.Println("=== Demo Complete ===")
}

// demoSystemInfo demonstrates system utilities.
func demoSystemInfo() {
	// Demo date
	fmt.Println("Current date:")
	dateFlags := &system.DateFlags{ISO: true}
	_ = system.Date(dateFlags, os.Stdout)

	// Demo whoami
	fmt.Println("\nCurrent user:")
	_ = system.Whoami(os.Stdout)

	// Demo uname
	fmt.Println("\nSystem information:")
	unameFlags := &system.UnameFlags{All: true}
	_ = system.Uname(unameFlags, os.Stdout)

	// Demo uptime
	fmt.Println("\nUptime:")
	_ = system.Uptime(os.Stdout)

	// Demo env
	fmt.Println("\nEnvironment (HOME):")
	home := os.Getenv("HOME")
	fmt.Println("HOME =", home)

	// Demo ps
	fmt.Println("\nProcess list:")
	psFlags := &system.PSFlags{All: true, NoHeader: true}
	_ = system.PS(psFlags, os.Stdout)

	// Demo df
	fmt.Println("\nDisk usage:")
	dfFlags := &system.DFFlags{Human: true}
	_ = system.DF(dfFlags, os.Stdout)
}

// demoTextProcessing demonstrates text utilities.
func demoTextProcessing() {
	fmt.Println("Text processing utilities are available in the text package:")
	fmt.Println("- grep: Pattern matching with regex support")
	fmt.Println("- sed: Stream editor for substitutions")
	fmt.Println("- wc: Word, line, and character counting")
	fmt.Println("- head/tail: Display file beginnings and endings")
	fmt.Println("- sort: Sort lines of text")
	fmt.Println("- uniq: Remove duplicate lines")
	fmt.Println("- cut: Extract columns from text")

	// Simple demonstration using standard output
	fmt.Println("\nExample wc output for this demo:")
	fmt.Println("(Simulated: 3 lines, 6 words, 42 characters)")
}
