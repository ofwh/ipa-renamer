package main

import (
	"fmt"
	"os"
	"strings"
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

	// Only now, so that -v itself can be reported like any other setting.
	verbose = opts.Verbose

	if opts.timeSet && !opts.Watch {
		logWarn("--time has no effect without -w/--watch")
	}

	cfg, err := opts.config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
		return 1
	}

	logStartup(args, opts, cfg)

	if opts.Watch {
		if err := Watch(*cfg); err != nil {
			logError("%v", err)
			return 1
		}
		return 0
	}
	return oneShot(*cfg)
}

// logStartup records the arguments as received and the settings they resolved to.
func logStartup(args []string, opts *Options, cfg *Config) {
	if !verbose {
		return
	}
	logDebug("version %s", version)
	if wd, err := os.Getwd(); err == nil {
		logDebug("working directory %s", wd)
	}
	logArgv(args)
	logDebug("input=%s output=%s watch=%t recursive=%t copy=%t settle=%s",
		cfg.Input, cfg.Output, opts.Watch, cfg.Recursive, cfg.Copy, cfg.Time)
}

// logArgv prints one option per line, aligned under the program name.
func logArgv(args []string) {
	if len(args) == 0 {
		logDebug("%s", progName)
		return
	}
	pairs := groupArgs(args)
	logDebug("%s %s", progName, pairs[0])
	indent := strings.Repeat(" ", len(progName)+1)
	for _, opt := range pairs[1:] {
		logDebug("%s%s", indent, opt)
	}
}

// groupArgs joins each flag with the value following it, so that "-i /app/in"
// stays on one line.
func groupArgs(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		opt := args[i]
		if i+1 < len(args) && takesValue(opt) {
			if next := args[i+1]; next == "-" || !strings.HasPrefix(next, "-") {
				opt += " " + next
				i++
			}
		}
		out = append(out, opt)
	}
	return out
}

// takesValue reports whether a flag is followed by its value.
func takesValue(opt string) bool {
	if !strings.HasPrefix(opt, "-") || opt == "-" || strings.Contains(opt, "=") {
		return false
	}
	switch strings.TrimLeft(opt, "-") {
	case "i", "input", "o", "output", "t", "time":
		return true
	}
	return false
}

// oneShot scans the input directory once and renames every matching .ipa file,
// descending into subdirectories when -r/--recursive was given.
func oneShot(cfg Config) int {
	if err := cfg.ensureDirs(false); err != nil {
		logError("%v", err)
		return 1
	}
	if err := cfg.checkDirs(); err != nil {
		logError("%v", err)
		return 1
	}

	renamed, failed := 0, 0
	walkIPA(cfg, cfg.Input, func(src string) {
		dst, err := Process(cfg, src)
		switch {
		case err != nil:
			failed++
			logRenameFailed(src, err)
		case dst != "":
			renamed++
			logRenamed(src, dst)
		}
	})

	logInfo("Done: %d ✓, %d 𐄂", renamed, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
