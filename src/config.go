package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Config carries the resolved runtime settings. Output and Time always carry a
// value once parseArgs applied its defaults; Input may be "." (current dir).
type Config struct {
	Input  string
	Output string
	Time   time.Duration
	Copy   bool
}

// config converts the directories to absolute paths so they remain valid
// regardless of the caller's working directory.
func (o *Options) config() (*Config, error) {
	input, err := filepath.Abs(o.Input)
	if err != nil {
		return nil, fmt.Errorf("resolve input dir %q: %w", o.Input, err)
	}
	output, err := filepath.Abs(o.Output)
	if err != nil {
		return nil, fmt.Errorf("resolve output dir %q: %w", o.Output, err)
	}
	return &Config{Input: input, Output: output, Time: o.Time, Copy: o.Copy}, nil
}

// ensureDirs creates Output and, when createInput is set, Input.
func (c *Config) ensureDirs(createInput bool) error {
	if err := os.MkdirAll(c.Output, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", c.Output, err)
	}
	if createInput {
		if err := os.MkdirAll(c.Input, 0o755); err != nil {
			return fmt.Errorf("create input directory %s: %w", c.Input, err)
		}
	}
	return nil
}

// checkDirs verifies that Input can be listed and Output can be written to, and
// logs the result for each directory.
func (c *Config) checkDirs() error {
	if err := checkReadableDir(c.Input); err != nil {
		return fmt.Errorf("input directory %s: %w", c.Input, err)
	}
	logInfo("input directory is readable: %s", c.Input)

	if err := checkWritableDir(c.Output); err != nil {
		return fmt.Errorf("output directory %s: %w", c.Output, err)
	}
	logInfo("output directory is writable: %s", c.Output)
	return nil
}

// checkReadableDir reports whether dir can be opened and listed.
func checkReadableDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Readdirnames(1); err != nil && err != io.EOF {
		return err
	}
	return nil
}

// checkWritableDir reports whether a file can be created in dir, and removes
// the probe file again.
func checkWritableDir(dir string) error {
	f, err := os.CreateTemp(dir, ".ipa-renamer-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}
