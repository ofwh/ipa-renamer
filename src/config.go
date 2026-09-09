package main

import (
	"fmt"
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
	return &Config{Input: input, Output: output, Time: o.Time}, nil
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
