package network

import (
	"bytes"
	"testing"
)

func TestParsePingFlags(t *testing.T) {
	flags, err := ParsePingFlags([]string{"-c", "10", "-i", "2", "google.com"})
	if err != nil {
		t.Errorf("ParsePingFlags returned error: %v", err)
	}
	if flags.Count != 10 {
		t.Errorf("Count = %d, want 10", flags.Count)
	}
	if flags.Interval != 2 {
		t.Errorf("Interval = %d, want 2", flags.Interval)
	}
}

func TestPing(t *testing.T) {
	flags := &PingFlags{Count: 1}
	var buf bytes.Buffer
	err := Ping("localhost", flags, &buf)
	if err != nil {
		t.Errorf("Ping returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Ping should produce output")
	}
}

func TestParseNetcatFlags(t *testing.T) {
	flags, args, err := ParseNetcatFlags([]string{"-l", "-p", "8080"})
	if err != nil {
		t.Errorf("ParseNetcatFlags returned error: %v", err)
	}
	if !flags.Listen {
		t.Error("Listen flag should be set")
	}
	if flags.Port != 8080 {
		t.Errorf("Port = %d, want 8080", flags.Port)
	}
	if len(args) != 0 {
		t.Errorf("Args = %v, want []", args)
	}
}

func TestParseCurlFlags(t *testing.T) {
	flags, args, err := ParseCurlFlags([]string{"-v", "-o", "output.html", "http://example.com"})
	if err != nil {
		t.Errorf("ParseCurlFlags returned error: %v", err)
	}
	if !flags.Verbose {
		t.Error("Verbose flag should be set")
	}
	if flags.Output != "output.html" {
		t.Errorf("Output = %s, want output.html", flags.Output)
	}
	if len(args) != 1 || args[0] != "http://example.com" {
		t.Errorf("Args = %v, want [http://example.com]", args)
	}
}

func TestParseWgetFlags(t *testing.T) {
	flags, args, err := ParseWgetFlags([]string{"-q", "-O", "output.html", "http://example.com/file.txt"})
	if err != nil {
		t.Errorf("ParseWgetFlags returned error: %v", err)
	}
	if !flags.Quiet {
		t.Error("Quiet flag should be set")
	}
	if flags.Output != "output.html" {
		t.Errorf("Output = %s, want output.html", flags.Output)
	}
	if len(args) != 1 || args[0] != "http://example.com/file.txt" {
		t.Errorf("Args = %v, want [http://example.com/file.txt]", args)
	}
}
