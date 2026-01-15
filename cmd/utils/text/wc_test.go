package text

import (
	"strings"
	"testing"
)

func TestParseWCFlags(t *testing.T) {
	flags, err := ParseWCFlags([]string{"-l", "-w", "file.txt"})
	if err != nil {
		t.Errorf("ParseWCFlags returned error: %v", err)
	}
	if !flags.Lines {
		t.Error("Lines flag should be set")
	}
	if !flags.Words {
		t.Error("Words flag should be set")
	}
	if len(flags.Files) != 1 || flags.Files[0] != "file.txt" {
		t.Errorf("Files = %v, want [file.txt]", flags.Files)
	}
}

func TestWCLines(t *testing.T) {
	input := "line one\nline two\nline three\n"
	reader := strings.NewReader(input)

	flags := &WCFlags{Lines: true}
	lines, words, _, _, err := WC(reader, flags)
	if err != nil {
		t.Errorf("WC returned error: %v", err)
	}
	if lines != 3 {
		t.Errorf("WC returned %d lines, want 3", lines)
	}
	if words != 6 {
		t.Errorf("WC returned %d words, want 6", words)
	}
}

func TestParseHeadFlags(t *testing.T) {
	flags, args, err := ParseHeadFlags([]string{"-n", "5", "file.txt"})
	if err != nil {
		t.Errorf("ParseHeadFlags returned error: %v", err)
	}
	if flags.Lines != 5 {
		t.Errorf("Lines = %d, want 5", flags.Lines)
	}
	if len(args) != 1 || args[0] != "file.txt" {
		t.Errorf("Args = %v, want [file.txt]", args)
	}
}

func TestParseTailFlags(t *testing.T) {
	flags, args, err := ParseTailFlags([]string{"-n", "10", "-f", "file.txt"})
	if err != nil {
		t.Errorf("ParseTailFlags returned error: %v", err)
	}
	if flags.Lines != 10 {
		t.Errorf("Lines = %d, want 10", flags.Lines)
	}
	if !flags.Follow {
		t.Error("Follow flag should be set")
	}
	if len(args) != 1 || args[0] != "file.txt" {
		t.Errorf("Args = %v, want [file.txt]", args)
	}
}

func TestParseSortFlags(t *testing.T) {
	flags, args, err := ParseSortFlags([]string{"-n", "-r", "file.txt"})
	if err != nil {
		t.Errorf("ParseSortFlags returned error: %v", err)
	}
	if !flags.Numeric {
		t.Error("Numeric flag should be set")
	}
	if !flags.Reverse {
		t.Error("Reverse flag should be set")
	}
	if len(args) != 1 || args[0] != "file.txt" {
		t.Errorf("Args = %v, want [file.txt]", args)
	}
}

func TestParseUniqFlags(t *testing.T) {
	flags, args, err := ParseUniqFlags([]string{"-c", "-d", "file.txt"})
	if err != nil {
		t.Errorf("ParseUniqFlags returned error: %v", err)
	}
	if !flags.Count {
		t.Error("Count flag should be set")
	}
	if !flags.Duplicate {
		t.Error("Duplicate flag should be set")
	}
	if len(args) != 1 || args[0] != "file.txt" {
		t.Errorf("Args = %v, want [file.txt]", args)
	}
}

func TestParseCutFlags(t *testing.T) {
	flags, args, err := ParseCutFlags([]string{"-d", ":", "-f", "1,3", "file.txt"})
	if err != nil {
		t.Errorf("ParseCutFlags returned error: %v", err)
	}
	if flags.Delimiter != ":" {
		t.Errorf("Delimiter = %s, want :", flags.Delimiter)
	}
	if flags.Fields != "1,3" {
		t.Errorf("Fields = %s, want 1,3", flags.Fields)
	}
	if len(args) != 1 || args[0] != "file.txt" {
		t.Errorf("Args = %v, want [file.txt]", args)
	}
}
