// Package flags provides common command-line flag parsing utilities
// for the core utilities, ensuring consistent flag handling across all commands.
package flags

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// ParseFlags parses the given flag set with the provided arguments.
// It returns the remaining non-flag arguments and any error.
func ParseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			// Help was requested, exit gracefully
			fmt.Fprintln(os.Stderr)
			fs.PrintDefaults()
			os.Exit(0)
		}
		return nil, err
	}
	return fs.Args(), nil
}

// ParseCommon parses common flags that most utilities support.
func ParseCommon() (recursive, force, verbose bool) {
	flag.Bool("R", false, "Recursively operate on directories")
	flag.Bool("r", false, "Recursively operate on directories")
	flag.Bool("f", false, "Force operation, ignore errors")
	flag.Bool("v", false, "Verbose output")
	return
}

// ModeFromString converts an octal mode string to os.FileMode.
func ModeFromString(s string) (os.FileMode, error) {
	var mode int64
	_, err := fmt.Sscanf(s, "%o", &mode)
	if err != nil {
		return 0, fmt.Errorf("invalid mode: %s", s)
	}
	return os.FileMode(mode), nil
}

// ParseNumericMode parses a numeric (octal) mode from string arguments.
// It handles modes like "755" or "+x" or "u+x".
func ParseNumericMode(args []string, defaultMode os.FileMode) (os.FileMode, error) {
	mode := defaultMode

	for _, arg := range args {
		// Skip options starting with -
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Try to parse as octal number
		if len(arg) >= 3 && len(arg) <= 4 {
			var val int64
			_, err := fmt.Sscanf(arg, "%o", &val)
			if err == nil {
				mode = os.FileMode(val)
				break
			}
		}
	}

	return mode, nil
}
