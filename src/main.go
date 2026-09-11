package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const progName = "ipa-renamer"

func main() {
	os.Exit(run(os.Args[1:]))
}

// run parses the command line and dispatches to one-shot or watch mode. It
// returns a process exit code.
func run(args []string) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
		fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", progName)
		return 2
	}
	if opts.ShowHelp {
		fmt.Print(usageText())
		return 0
	}
	if opts.ShowVersion {
		fmt.Printf("%s %s\n", progName, version)
		return 0
	}

	if opts.timeSet && !opts.Watch {
		logWarn("--time has no effect without -w/--watch")
	}

	cfg, err := opts.config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
		return 1
	}

	if opts.Watch {
		if err := Watch(*cfg); err != nil {
			logError("%v", err)
			return 1
		}
		return 0
	}
	return oneShot(*cfg)
}

// oneShot scans the input directory once and renames every matching .ipa file.
func oneShot(cfg Config) int {
	if err := cfg.ensureDirs(false); err != nil {
		logError("%v", err)
		return 1
	}
	if err := cfg.checkDirs(); err != nil {
		logError("%v", err)
		return 1
	}
	entries, err := os.ReadDir(cfg.Input)
	if err != nil {
		logError("read input directory: %v", err)
		return 1
	}

	renamed, failed := 0, 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(cfg.Input, entry.Name())
		dst, err := Process(cfg, src)
		switch {
		case err != nil:
			failed++
			logRenameFailed(src, err)
		case dst != "":
			renamed++
			logRenamed(src, dst)
		}
	}

	logInfo("Done: %d ✓, %d 𐄂", renamed, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
