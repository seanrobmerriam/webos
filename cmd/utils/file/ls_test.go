package file

import (
	"os"
	"testing"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{100, "100"},
		{1023, "1023"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1048576, "1.0M"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.input)
		if result != tt.expected {
			t.Errorf("FormatSize(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatMode(t *testing.T) {
	tests := []struct {
		mode     os.FileMode
		expected string
	}{
		{0755, "drwxr-xr-x"},
		{0644, "-rw-r--r--"},
		{0555, "-r-xr-xr-x"},
		{0444, "-r--r--r--"},
		{0000, "----------"},
	}

	for _, tt := range tests {
		result := FormatMode(tt.mode)
		if len(result) != 10 {
			t.Errorf("FormatMode(%o) = %s, wrong length", tt.mode, result)
		}
	}
}

func TestParseLSFlags(t *testing.T) {
	// Test basic flag parsing
	flags, args, err := ParseLSFlags([]string{"-l", "-a", "/tmp"})
	if err != nil {
		t.Errorf("ParseLSFlags returned error: %v", err)
	}
	if !flags.Long {
		t.Error("Long flag should be set")
	}
	if !flags.All {
		t.Error("All flag should be set")
	}
	if len(args) != 1 || args[0] != "/tmp" {
		t.Errorf("Args = %v, want [/tmp]", args)
	}
}

func TestParseLSFlagsDefaults(t *testing.T) {
	flags, args, err := ParseLSFlags([]string{})
	if err != nil {
		t.Errorf("ParseLSFlags returned error: %v", err)
	}
	if flags.Long {
		t.Error("Long flag should not be set by default")
	}
	if len(args) != 0 {
		t.Errorf("Args = %v, want []", args)
	}
}
