package main

import (
	"testing"
	"time"
)

func TestParseArgsDefaults(t *testing.T) {
	o, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs(nil) error: %v", err)
	}
	if o.Input != "." {
		t.Errorf("default input = %q, want %q", o.Input, ".")
	}
	if o.Output != defaultOutputDir {
		t.Errorf("default output = %q, want %q", o.Output, defaultOutputDir)
	}
	if o.Time != defaultTimeSeconds*time.Second {
		t.Errorf("default time = %v, want %v", o.Time, defaultTimeSeconds*time.Second)
	}
	if o.Watch || o.ShowHelp || o.ShowVersion {
		t.Errorf("flags unexpectedly set: watch=%v help=%v version=%v", o.Watch, o.ShowHelp, o.ShowVersion)
	}
}

func TestParseArgsPositionalInput(t *testing.T) {
	o, err := parseArgs([]string{"/path/to/input", "-o", "/path/to/output"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if o.Input != "/path/to/input" {
		t.Errorf("input = %q, want %q", o.Input, "/path/to/input")
	}
	if o.Output != "/path/to/output" {
		t.Errorf("output = %q, want %q", o.Output, "/path/to/output")
	}
}

func TestParseArgsSameInputBothWays(t *testing.T) {
	if _, err := parseArgs([]string{"-i", "/in", "/in"}); err != nil {
		t.Errorf("identical -i and positional should be allowed, got error: %v", err)
	}
}

func TestParseArgsInputConflictInAnyOrder(t *testing.T) {
	for _, args := range [][]string{
		{"-i", "/in/a", "/in/b"},
		{"/in/b", "-i", "/in/a"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%v): conflicting -i and positional input should error", args)
		}
	}
}

func TestParseArgsTooManyPositionals(t *testing.T) {
	if _, err := parseArgs([]string{"/in/a", "/in/b"}); err == nil {
		t.Error("two positional arguments should error")
	}
}

func TestParseArgsShortAndLongForms(t *testing.T) {
	cases := []struct {
		args  []string
		watch bool
		time  time.Duration
	}{
		{[]string{"-i", "x", "-o", "y", "-w"}, true, 5 * time.Second},
		{[]string{"--input", "x", "--output", "y", "--watch"}, true, 5 * time.Second},
		{[]string{"--input=x", "--output=y"}, false, 5 * time.Second},
		{[]string{"-i=x", "-o=y"}, false, 5 * time.Second},
		{[]string{"-i", "x", "-o", "y", "-t", "12", "-w"}, true, 12 * time.Second},
		{[]string{"--input", "x", "--output", "y", "--time", "30", "--watch"}, true, 30 * time.Second},
		{[]string{"-i=x", "-o=y", "-t=20", "-w"}, true, 20 * time.Second},
	}
	for _, c := range cases {
		o, err := parseArgs(c.args)
		if err != nil {
			t.Fatalf("parseArgs(%v) error: %v", c.args, err)
		}
		if o.Input != "x" || o.Output != "y" || o.Watch != c.watch || o.Time != c.time {
			t.Errorf("parseArgs(%v) = {input:%q output:%q watch:%v time:%v}", c.args, o.Input, o.Output, o.Watch, o.Time)
		}
	}
}

func TestParseArgsTimeValidation(t *testing.T) {
	for _, args := range [][]string{
		{"-t", "0"}, {"-t", "-3"}, {"-t", "abc"}, {"--time=1.5"}, {"-t"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%v): invalid time should error", args)
		}
	}
}

func TestParseArgsHelpVersion(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"-V"}, {"--version"}} {
		o, err := parseArgs(args)
		if err != nil {
			t.Fatalf("parseArgs(%v) error: %v", args, err)
		}
		expectHelp := args[0] == "-h" || args[0] == "--help"
		if o.ShowHelp != expectHelp || o.ShowVersion == expectHelp {
			t.Errorf("parseArgs(%v): help=%v version=%v", args, o.ShowHelp, o.ShowVersion)
		}
	}
}

func TestParseArgsUnknownFlag(t *testing.T) {
	if _, err := parseArgs([]string{"-x"}); err == nil {
		t.Error("unknown flag should error")
	}
	if _, err := parseArgs([]string{"--bogus"}); err == nil {
		t.Error("unknown long flag should error")
	}
}

func TestParseArgsMissingValue(t *testing.T) {
	if _, err := parseArgs([]string{"-o"}); err == nil {
		t.Error("flag missing a value should error")
	}
	if _, err := parseArgs([]string{"--output="}); err == nil {
		t.Error("empty inline value should error")
	}
}

func TestParseArgsWatchTakesNoValue(t *testing.T) {
	if _, err := parseArgs([]string{"-w=true"}); err == nil {
		t.Error("-w=true should error (watch takes no value)")
	}
}

func TestParseArgsIgnoresEnv(t *testing.T) {
	// Parameter defaults are compiled in and never overridden by environment
	// variables; set some to prove they are ignored.
	t.Setenv("IPA_RENAMER_INPUT", "/env/in")
	t.Setenv("IPA_RENAMER_OUTPUT", "/env/out")
	t.Setenv("IPA_RENAMER_TIME", "17")

	o, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if o.Input != defaultInputDir || o.Output != defaultOutputDir || o.Time != defaultTimeSeconds*time.Second {
		t.Errorf("env vars must be ignored: input=%q output=%q time=%v", o.Input, o.Output, o.Time)
	}
}
