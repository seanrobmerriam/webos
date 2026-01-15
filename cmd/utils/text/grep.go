// Package text provides text processing utilities: grep, sed, awk, cut, sort, uniq, wc, head, tail.
package text

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
)

// GrepFlags holds command-line flags for grep.
type GrepFlags struct {
	IgnoreCase bool // Case-insensitive matching
	Invert     bool // Invert match
	LineNum    bool // Show line numbers
	Quiet      bool // Suppress output
	Count      bool // Show only count of matches
	Recursive  bool // Recursive search
	Fixed      bool // Fixed string matching (not regex)
	WholeLine  bool // Match whole line
}

// ParseGrepFlags parses command-line flags for grep.
func ParseGrepFlags(args []string) (*GrepFlags, []string, error) {
	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: grep [OPTIONS] PATTERN [FILE...]

Search for patterns in files.

Options:
`)
		fs.PrintDefaults()
	}

	ignoreCase := fs.Bool("i", false, "Ignore case distinctions")
	invert := fs.Bool("v", false, "Invert match")
	lineNum := fs.Bool("n", false, "Show line numbers")
	quiet := fs.Bool("q", false, "Quiet mode")
	count := fs.Bool("c", false, "Count matching lines")
	recursive := fs.Bool("r", false, "Recursive search")
	fixed := fs.Bool("F", false, "Fixed strings")
	wholeLine := fs.Bool("x", false, "Match whole lines")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &GrepFlags{
		IgnoreCase: *ignoreCase,
		Invert:     *invert,
		LineNum:    *lineNum,
		Quiet:      *quiet,
		Count:      *count,
		Recursive:  *recursive,
		Fixed:      *fixed,
		WholeLine:  *wholeLine,
	}

	return flags, fs.Args(), nil
}

// Grep searches for patterns in input.
func Grep(reader io.Reader, pattern string, flags *GrepFlags) error {
	var regex *regexp.Regexp
	var err error

	if flags.Fixed {
		pattern = regexp.QuoteMeta(pattern)
	}

	if flags.IgnoreCase {
		pattern = "(?i)" + pattern
	}

	if flags.WholeLine {
		pattern = "^" + pattern + "$"
	}

	regex, err = regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %v", err)
	}

	scanner := bufio.NewScanner(reader)
	lineNum := 0
	matchCount := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		matched := regex.MatchString(line)

		if flags.Invert {
			matched = !matched
		}

		if matched {
			matchCount++
			if flags.Quiet {
				return nil
			}
			if flags.Count {
				continue
			}
			if flags.LineNum {
				fmt.Printf("%d:%s\n", lineNum, line)
			} else {
				fmt.Println(line)
			}
		}
	}

	if flags.Count {
		fmt.Println(matchCount)
	}

	return scanner.Err()
}

// GrepFile searches for pattern in a file.
func GrepFile(path, pattern string, flags *GrepFlags) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	defer file.Close()

	return Grep(file, pattern, flags)
}
