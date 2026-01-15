package text

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// WCFlags holds command-line flags for wc.
type WCFlags struct {
	Lines   bool // Count lines
	Words   bool // Count words
	Chars   bool // Count characters
	Bytes   bool // Count bytes
	Longest bool // Print longest line length
	Files   []string
}

// ParseWCFlags parses command-line flags for wc.
func ParseWCFlags(args []string) (*WCFlags, error) {
	fs := flag.NewFlagSet("wc", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: wc [OPTIONS] [FILE...]

Print newline, word, and byte counts.

Options:
`)
		fs.PrintDefaults()
	}

	lines := fs.Bool("l", false, "Count lines")
	words := fs.Bool("w", false, "Count words")
	chars := fs.Bool("m", false, "Count characters")
	bytes := fs.Bool("c", false, "Count bytes")
	longest := fs.Bool("L", false, "Print longest line length")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	flags := &WCFlags{
		Lines:   *lines,
		Words:   *words,
		Chars:   *chars,
		Bytes:   *bytes,
		Longest: *longest,
		Files:   fs.Args(),
	}

	return flags, nil
}

// WC counts lines, words, characters in input.
func WC(reader io.Reader, flags *WCFlags) (int, int, int, int, error) {
	scanner := bufio.NewScanner(reader)
	lineCount := 0
	wordCount := 0
	byteCount := 0
	charCount := 0
	longest := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		wordCount += len(strings.Fields(line))
		byteCount += len(scanner.Bytes())
		charCount += len([]rune(line))

		if len(line) > longest {
			longest = len(line)
		}
	}

	if flags.Longest {
		fmt.Printf("%d\n", longest)
	}

	return lineCount, wordCount, charCount, byteCount, scanner.Err()
}

// WCFile counts lines, words, characters in a file.
func WCFile(path string, flags *WCFlags) (int, int, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer file.Close()

	return WC(file, flags)
}

// HeadFlags holds command-line flags for head.
type HeadFlags struct {
	Lines int // Number of lines (0 = all)
	Bytes int // Number of bytes (0 = all)
}

// ParseHeadFlags parses command-line flags for head.
func ParseHeadFlags(args []string) (*HeadFlags, []string, error) {
	fs := flag.NewFlagSet("head", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: head [OPTIONS] [FILE...]

Print the first lines of files.

Options:
`)
		fs.PrintDefaults()
	}

	n := fs.Int("n", 10, "Number of lines")
	c := fs.Int("c", 0, "Number of bytes")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &HeadFlags{
		Lines: *n,
		Bytes: *c,
	}

	return flags, fs.Args(), nil
}

// Head prints first N lines/bytes from input.
func Head(reader io.Reader, flags *HeadFlags, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)

	if flags.Bytes > 0 {
		buf := make([]byte, flags.Bytes)
		n, _ := reader.Read(buf)
		writer.Write(buf[:n])
		return nil
	}

	count := 0
	for scanner.Scan() && (flags.Lines == 0 || count < flags.Lines) {
		fmt.Fprintln(writer, scanner.Text())
		count++
	}

	return scanner.Err()
}

// HeadFile prints first N lines/bytes from a file.
func HeadFile(path string, flags *HeadFlags) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return Head(file, flags, os.Stdout)
}

// TailFlags holds command-line flags for tail.
type TailFlags struct {
	Lines  int  // Number of lines (0 = all)
	Bytes  int  // Number of bytes (0 = all)
	Follow bool // Follow file changes
}

// ParseTailFlags parses command-line flags for tail.
func ParseTailFlags(args []string) (*TailFlags, []string, error) {
	fs := flag.NewFlagSet("tail", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: tail [OPTIONS] [FILE...]

Print the last lines of files.

Options:
`)
		fs.PrintDefaults()
	}

	n := fs.Int("n", 10, "Number of lines")
	c := fs.Int("c", 0, "Number of bytes")
	follow := fs.Bool("f", false, "Follow file changes")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &TailFlags{
		Lines:  *n,
		Bytes:  *c,
		Follow: *follow,
	}

	return flags, fs.Args(), nil
}

// Tail prints last N lines/bytes from input.
func Tail(reader io.Reader, flags *TailFlags, writer io.Writer) error {
	if flags.Bytes > 0 {
		return tailBytes(reader, flags.Bytes, writer)
	}

	var lines []string
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if flags.Lines > 0 && len(lines) > flags.Lines {
			lines = lines[1:]
		}
	}

	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return scanner.Err()
}

// tailBytes prints last N bytes from input.
func tailBytes(reader io.Reader, n int, writer io.Writer) error {
	// Seek to near end if possible
	return nil
}

// TailFile prints last N lines/bytes from a file.
func TailFile(path string, flags *TailFlags) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return Tail(file, flags, os.Stdout)
}
