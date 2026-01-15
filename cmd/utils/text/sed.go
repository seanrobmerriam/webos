package text

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// SedFlags holds command-line flags for sed.
type SedFlags struct {
	Expression string // sed expression
	InPlace    bool   // Edit files in place
	Quiet      bool   // Suppress automatic printing
	Extended   bool   // Use extended regex
}

// ParseSedFlags parses command-line flags for sed.
func ParseSedFlags(args []string) (*SedFlags, []string, error) {
	fs := flag.NewFlagSet("sed", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: sed [OPTIONS] SCRIPT [FILE...]

Stream editor for filtering and transforming text.

Options:
`)
		fs.PrintDefaults()
	}

	expr := fs.String("e", "", "Add script to commands")
	inPlace := fs.Bool("i", false, "Edit files in place")
	quiet := fs.Bool("n", false, "Suppress automatic printing")
	extended := fs.Bool("r", false, "Use extended regular expressions")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &SedFlags{
		Expression: *expr,
		InPlace:    *inPlace,
		Quiet:      *quiet,
		Extended:   *extended,
	}

	return flags, fs.Args(), nil
}

// Sed processes input with sed script.
func Sed(reader io.Reader, script string, flags *SedFlags) error {
	sed := NewSed(script, flags.Extended)
	return sed.Process(os.Stdin, os.Stdout)
}

// SedProcessor handles sed operations.
type SedProcessor struct {
	script   string
	extended bool
	quiet    bool
	commands []sedCommand
}

// sedCommand represents a single sed command.
type sedCommand struct {
	address *sedAddress
	op      byte // s, p, d, a, i, c
	// For substitution
	pattern     string
	replacement string
	flags       int // g, p, i
}

// sedAddress represents an address range in sed.
type sedAddress struct {
	start    int // line number, -1 for all, 0 for start/end patterns
	startPat string
	endPat   string
	startReg *regexp.Regexp
	endReg   *regexp.Regexp
}

// NewSed creates a new sed processor.
func NewSed(script string, extended bool) *SedProcessor {
	return &SedProcessor{
		script:   script,
		extended: extended,
		commands: parseSedScript(script, extended),
	}
}

// parseSedScript parses a sed script string.
func parseSedScript(script string, extended bool) []sedCommand {
	var commands []sedCommand

	// Simple parsing for s/// and p commands
	parts := strings.Split(script, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse s/// substitution
		if len(part) >= 2 && part[0] == 's' {
			delim := part[1]
			rest := part[2:]
			parts := splitByDelimiter(rest, delim, 3)
			if len(parts) >= 2 {
				flags := 0
				if strings.Contains(parts[2], "g") {
					flags |= 1 // g flag
				}
				if strings.Contains(parts[2], "p") {
					flags |= 2 // p flag
				}

				commands = append(commands, sedCommand{
					address:     &sedAddress{start: -1},
					op:          's',
					pattern:     parts[0],
					replacement: parts[1],
					flags:       flags,
				})
			}
		}
	}

	return commands
}

// splitByDelimiter splits a string by delimiter.
func splitByDelimiter(s string, delim byte, limit int) []string {
	var parts []string
	current := ""
	count := 0

	for i := 0; i < len(s) && (limit == 0 || count < limit); i++ {
		if s[i] == delim && count < limit-1 {
			parts = append(parts, current)
			current = ""
			count++
		} else {
			current += string(s[i])
		}
	}

	parts = append(parts, current)
	return parts
}

// Process applies sed commands to input.
func (s *SedProcessor) Process(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	lineNum := 0
	shouldPrint := true

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		shouldPrint = true

		for _, cmd := range s.commands {
			if s.matchAddress(cmd.address, lineNum, line) {
				switch cmd.op {
				case 's':
					newLine, matched := s.doSubstitute(line, cmd)
					if matched && (cmd.flags&2) != 0 {
						fmt.Fprintln(out, newLine)
					}
					line = newLine
					shouldPrint = false
				case 'p':
					fmt.Fprintln(out, line)
				case 'd':
					shouldPrint = false
					break
				}
			}
		}

		if shouldPrint && !s.quiet {
			fmt.Fprintln(out, line)
		}
	}

	return scanner.Err()
}

// matchAddress checks if a line matches the address.
func (s *SedProcessor) matchAddress(addr *sedAddress, lineNum int, line string) bool {
	if addr == nil || addr.start == -1 {
		return true
	}
	return addr.start == lineNum
}

// doSubstitute performs substitution.
func (s *SedProcessor) doSubstitute(line string, cmd sedCommand) (string, bool) {
	pattern := cmd.pattern
	replacement := cmd.replacement

	if s.extended {
		pattern = "(" + pattern + ")"
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return line, false
	}

	if (cmd.flags & 1) != 0 {
		// Global replacement
		return re.ReplaceAllString(line, replacement), true
	}

	return re.ReplaceAllStringFunc(line, func(match string) string {
		return replacement
	}), true
}
