package system

import (
	"bytes"
	"os"
	"testing"
)

func TestParsePSFlags(t *testing.T) {
	flags, err := ParsePSFlags([]string{"-a", "-f"})
	if err != nil {
		t.Errorf("ParsePSFlags returned error: %v", err)
	}
	if !flags.All {
		t.Error("All flag should be set")
	}
	if !flags.Full {
		t.Error("Full flag should be set")
	}
}

func TestParsePSFlagsDefaults(t *testing.T) {
	flags, err := ParsePSFlags([]string{})
	if err != nil {
		t.Errorf("ParsePSFlags returned error: %v", err)
	}
	if flags.All {
		t.Error("All flag should not be set by default")
	}
	if flags.Full {
		t.Error("Full flag should not be set by default")
	}
}

func TestPS(t *testing.T) {
	flags := &PSFlags{All: true, NoHeader: true}
	var buf bytes.Buffer
	err := PS(flags, &buf)
	if err != nil {
		t.Errorf("PS returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("PS should produce output")
	}
}

func TestParseDFFlags(t *testing.T) {
	flags, err := ParseDFFlags([]string{"-h"})
	if err != nil {
		t.Errorf("ParseDFFlags returned error: %v", err)
	}
	if !flags.Human {
		t.Error("Human flag should be set")
	}
}

func TestDF(t *testing.T) {
	flags := &DFFlags{Human: true}
	var buf bytes.Buffer
	err := DF(flags, &buf)
	if err != nil {
		t.Errorf("DF returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("DF should produce output")
	}
}

func TestParseDUFlags(t *testing.T) {
	flags, args, err := ParseDUFlags([]string{"-h", "-s", "/tmp"})
	if err != nil {
		t.Errorf("ParseDUFlags returned error: %v", err)
	}
	if !flags.Human {
		t.Error("Human flag should be set")
	}
	if !flags.Summary {
		t.Error("Summary flag should be set")
	}
	if len(args) != 1 || args[0] != "/tmp" {
		t.Errorf("Args = %v, want [/tmp]", args)
	}
}

func TestDU(t *testing.T) {
	flags := &DUFlags{Human: true}
	var buf bytes.Buffer
	err := DU([]string{"/tmp"}, flags, &buf)
	if err != nil {
		t.Errorf("DU returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("DU should produce output")
	}
}

func TestParseUnameFlags(t *testing.T) {
	flags, err := ParseUnameFlags([]string{"-a"})
	if err != nil {
		t.Errorf("ParseUnameFlags returned error: %v", err)
	}
	if !flags.All {
		t.Error("All flag should be set")
	}
}

func TestUname(t *testing.T) {
	flags := &UnameFlags{All: true}
	var buf bytes.Buffer
	err := Uname(flags, &buf)
	if err != nil {
		t.Errorf("Uname returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Uname should produce output")
	}
}

func TestParseDateFlags(t *testing.T) {
	flags, err := ParseDateFlags([]string{"-I"})
	if err != nil {
		t.Errorf("ParseDateFlags returned error: %v", err)
	}
	if !flags.ISO {
		t.Error("ISO flag should be set")
	}
}

func TestDate(t *testing.T) {
	flags := &DateFlags{ISO: true}
	var buf bytes.Buffer
	err := Date(flags, &buf)
	if err != nil {
		t.Errorf("Date returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Date should produce output")
	}
}

func TestUptime(t *testing.T) {
	var buf bytes.Buffer
	err := Uptime(&buf)
	if err != nil {
		t.Errorf("Uptime returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Uptime should produce output")
	}
}

func TestWhoami(t *testing.T) {
	var buf bytes.Buffer
	err := Whoami(&buf)
	if err != nil {
		t.Errorf("Whoami returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Whoami should produce output")
	}
	expected := os.Getenv("USER")
	if buf.String() != expected+"\n" {
		t.Errorf("Whoami = %s, want %s", buf.String(), expected)
	}
}

func TestEnv(t *testing.T) {
	var buf bytes.Buffer
	err := Env(&buf)
	if err != nil {
		t.Errorf("Env returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Env should produce output")
	}
}

func TestSetEnv(t *testing.T) {
	err := SetEnv("WEBOS_TEST_VAR", "test_value")
	if err != nil {
		t.Errorf("SetEnv returned error: %v", err)
	}
	if os.Getenv("WEBOS_TEST_VAR") != "test_value" {
		t.Error("Environment variable should be set")
	}
}

func TestUnsetEnv(t *testing.T) {
	os.Setenv("WEBOS_TEST_VAR", "test_value")
	err := UnsetEnv("WEBOS_TEST_VAR")
	if err != nil {
		t.Errorf("UnsetEnv returned error: %v", err)
	}
	if os.Getenv("WEBOS_TEST_VAR") != "" {
		t.Error("Environment variable should be unset")
	}
}
