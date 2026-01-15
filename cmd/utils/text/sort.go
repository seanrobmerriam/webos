package text

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// SortFlags holds command-line flags for sort.
type SortFlags struct {
	Numeric    bool   // Sort numerically
	Reverse    bool   // Reverse order
	IgnoreCase bool   // Ignore case
	Unique     bool   // Remove duplicates
	Delimiter  string // Field delimiter
	Key        int    // Sort by key field
	MonthSort  bool   // Sort by month name
}

// ParseSortFlags parses command-line flags for sort.
func ParseSortFlags(args []string) (*SortFlags, []string, error) {
	fs := flag.NewFlagSet("sort", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: sort [OPTIONS] [FILE...]

Sort lines of text.

Options:
`)
		fs.PrintDefaults()
	}

	numeric := fs.Bool("n", false, "Sort numerically")
	reverse := fs.Bool("r", false, "Reverse sort")
	ignoreCase := fs.Bool("f", false, "Ignore case")
	unique := fs.Bool("u", false, "Unique lines only")
	delim := fs.String("t", "", "Field delimiter")
	key := fs.Int("k", 0, "Sort by key field")
	month := fs.Bool("M", false, "Sort by month")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &SortFlags{
		Numeric:    *numeric,
		Reverse:    *reverse,
		IgnoreCase: *ignoreCase,
		Unique:     *unique,
		Delimiter:  *delim,
		Key:        *key,
		MonthSort:  *month,
	}

	return flags, fs.Args(), nil
}

// Sort sorts lines from input.
func Sort(reader io.Reader, flags *SortFlags, writer io.Writer) error {
	var lines []string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	sort.Slice(lines, func(i, j int) bool {
		a, b := lines[i], lines[j]

		// Extract sort key
		if flags.Key > 0 && flags.Delimiter != "" {
			a = extractField(a, flags.Key, flags.Delimiter)
			b = extractField(b, flags.Key, flags.Delimiter)
		}

		if flags.Numeric {
			// Try numeric comparison
			af, aok := parseFloat(a)
			bf, bok := parseFloat(b)
			if aok && bok {
				if flags.Reverse {
					return af > bf
				}
				return af < bf
			}
		}

		if flags.IgnoreCase {
			a = strings.ToLower(a)
			b = strings.ToLower(b)
		}

		if flags.Reverse {
			return a > b
		}
		return a < b
	})

	if flags.Unique {
		var uniqueLines []string
		prev := ""
		for _, line := range lines {
			if line != prev {
				uniqueLines = append(uniqueLines, line)
				prev = line
			}
		}
		lines = uniqueLines
	}

	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return nil
}

// extractField extracts a field by number (1-indexed) using delimiter.
func extractField(line string, field int, delim string) string {
	fields := strings.Split(line, delim)
	if field <= 0 || field > len(fields) {
		return line
	}
	return fields[field-1]
}

// parseFloat attempts to parse a string as float.
func parseFloat(s string) (float64, bool) {
	var f float64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f, err == nil
}

// UniqFlags holds command-line flags for uniq.
type UniqFlags struct {
	Count      bool // Show counts
	Duplicate  bool // Only show duplicate lines
	Unique     bool // Only show unique lines
	IgnoreCase bool // Ignore case
}

// ParseUniqFlags parses command-line flags for uniq.
func ParseUniqFlags(args []string) (*UniqFlags, []string, error) {
	fs := flag.NewFlagSet("uniq", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: uniq [OPTIONS] [FILE...]

Report or omit repeated lines.

Options:
`)
		fs.PrintDefaults()
	}

	count := fs.Bool("c", false, "Prefix lines by number of occurrences")
	duplicate := fs.Bool("d", false, "Only print duplicate lines")
	unique := fs.Bool("u", false, "Only print unique lines")
	ignoreCase := fs.Bool("i", false, "Ignore case")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &UniqFlags{
		Count:      *count,
		Duplicate:  *duplicate,
		Unique:     *unique,
		IgnoreCase: *ignoreCase,
	}

	return flags, fs.Args(), nil
}

// Uniq prints unique lines from input.
func Uniq(reader io.Reader, flags *UniqFlags, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	var prev string
	count := 0
	for _, line := range lines {
		if flags.IgnoreCase && strings.EqualFold(line, prev) {
			count++
			continue
		}

		if prev != "" {
			if err := printUniqLine(prev, count, flags, writer); err != nil {
				return err
			}
		}

		prev = line
		count = 1
	}

	if prev != "" {
		if err := printUniqLine(prev, count, flags, writer); err != nil {
			return err
		}
	}

	return nil
}

// printUniqLine prints a line based on uniq flags.
func printUniqLine(line string, count int, flags *UniqFlags, writer io.Writer) error {
	switch {
	case flags.Count:
		fmt.Fprintf(writer, "%d %s\n", count, line)
	case flags.Duplicate:
		if count > 1 {
			fmt.Fprintln(writer, line)
		}
	case flags.Unique:
		if count == 1 {
			fmt.Fprintln(writer, line)
		}
	default:
		fmt.Fprintln(writer, line)
	}
	return nil
}

// CutFlags holds command-line flags for cut.
type CutFlags struct {
	Delimiter string // Field delimiter
	Fields    string // Field specification (e.g., "1,3" or "1-3")
	Bytes     string // Byte specification
	Chars     string // Character specification
}

// ParseCutFlags parses command-line flags for cut.
func ParseCutFlags(args []string) (*CutFlags, []string, error) {
	fs := flag.NewFlagSet("cut", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: cut [OPTIONS] [FILE...]

Remove sections from each line of files.

Options:
`)
		fs.PrintDefaults()
	}

	delim := fs.String("d", "\t", "Field delimiter")
	fields := fs.String("f", "", "Field specification")
	bytes := fs.String("b", "", "Byte specification")
	chars := fs.String("c", "", "Character specification")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &CutFlags{
		Delimiter: *delim,
		Fields:    *fields,
		Bytes:     *bytes,
		Chars:     *chars,
	}

	return flags, fs.Args(), nil
}

// Cut extracts sections from each line of input.
func Cut(reader io.Reader, flags *CutFlags, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if flags.Fields != "" {
			parts := strings.Split(line, flags.Delimiter)
			indices := parseFieldSpec(flags.Fields)

			var result []string
			for _, idx := range indices {
				if idx >= 0 && idx < len(parts) {
					result = append(result, parts[idx])
				}
			}

			if len(result) > 0 {
				fmt.Fprintln(writer, strings.Join(result, flags.Delimiter))
			}
		} else if flags.Chars != "" {
			indices := parseRangeSpec(flags.Chars)
			var result []byte
			runes := []rune(line)
			for _, idx := range indices {
				if idx >= 0 && idx < len(runes) {
					result = append(result, byte(runes[idx]))
				}
			}
			writer.Write(result)
			fmt.Fprintln(writer)
		} else {
			fmt.Fprintln(writer, line)
		}
	}

	return scanner.Err()
}

// parseFieldSpec parses a field specification like "1,3" or "1-3".
func parseFieldSpec(spec string) []int {
	var indices []int
	parts := strings.Split(spec, ",")
	for _, part := range parts {
		if idx := parseFieldNum(part); idx >= 0 {
			indices = append(indices, idx)
		}
	}
	return indices
}

// parseFieldNum parses a single field number.
func parseFieldNum(s string) int {
	s = strings.TrimSpace(s)
	if n, err := fmt.Sscanf(s, "%d", new(int)); n == 1 && err == nil {
		var result int
		fmt.Sscanf(s, "%d", &result)
		if result > 0 {
			return result - 1 // Convert to 0-indexed
		}
	}
	return -1
}

// parseRangeSpec parses a range specification like "1-5" or "1,3-5".
func parseRangeSpec(spec string) []int {
	var indices []int
	parts := strings.Split(spec, ",")
	for _, part := range parts {
		indices = append(indices, parseRange(part)...)
	}
	return indices
}

// parseRange parses a range like "1-5".
func parseRange(s string) []int {
	s = strings.TrimSpace(s)
	var start, end int
	_, err := fmt.Sscanf(s, "%d-%d", &start, &end)
	if err == nil && start > 0 && end > 0 && start <= end {
		var indices []int
		for i := start; i <= end; i++ {
			indices = append(indices, i-1) // Convert to 0-indexed
		}
		return indices
	}

	// Single number
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil && n > 0 {
		return []int{n - 1}
	}

	return nil
}
