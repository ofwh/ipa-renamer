package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// Compiled-in defaults, used when no value is supplied on the command line.
	defaultInputDir    = "."
	defaultTimeSeconds = 5
)

// Options holds the parsed command line.
type Options struct {
	Input       string        // input directory (default: defaultInputDir)
	Output      string        // output directory (default: Input)
	Time        time.Duration // watch settle delay (default: defaultTimeSeconds * time.Second)
	Watch       bool          // watch Input for new/changed .ipa files
	Copy        bool          // copy instead of renaming, keeping the source file
	ShowHelp    bool
	ShowVersion bool

	inputFlag string // input given via -i/--input (used to check positional conflicts)
	timeSet   bool   // whether -t/--time was given explicitly
}

func flagUsage(name string, long bool) string {
	if long {
		return "--" + name
	}
	return "-" + name
}

// parseSeconds validates a positive number of seconds.
func parseSeconds(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid value %q: expected a positive number of seconds", s)
	}
	return n, nil
}

// parseArgs parses the command line. Flags and positional arguments may be
// freely mixed; -o and --output are equivalent, and a value may be given either
// as "-o value" or "-o=value".
func parseArgs(args []string) (*Options, error) {
	o := &Options{}
	var positional []string

	outputGiven := false

	i := 0
	for i < len(args) {
		arg := args[i]

		// "--" ends option parsing; the rest are positional.
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		// A lone "-" or any token that does not start with "-" is positional.
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			i++
			continue
		}

		// Strip leading dashes; "--watch" and "-w" reduce to "watch" and "w".
		rest := arg[1:]
		long := strings.HasPrefix(rest, "-")
		if long {
			rest = rest[1:]
		}

		name := rest
		hasInline := false
		inline := ""
		if eq := strings.IndexByte(rest, '='); eq >= 0 {
			name = rest[:eq]
			inline = rest[eq+1:]
			hasInline = true
		}
		if name == "" {
			return nil, fmt.Errorf("invalid flag %q", arg)
		}

		noValue := func() error {
			if hasInline {
				return fmt.Errorf("flag %s does not take a value", flagUsage(name, long))
			}
			return nil
		}
		// takeValue returns the flag value from either "-o=x" or "-o x".
		takeValue := func() (string, error) {
			if hasInline {
				if inline == "" {
					return "", fmt.Errorf("flag %s requires a value", flagUsage(name, long))
				}
				return inline, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("flag %s requires a value", flagUsage(name, long))
			}
			return args[i], nil
		}

		switch name {
		case "h", "help":
			if err := noValue(); err != nil {
				return nil, err
			}
			o.ShowHelp = true
		case "V", "version":
			if err := noValue(); err != nil {
				return nil, err
			}
			o.ShowVersion = true
		case "w", "watch":
			if err := noValue(); err != nil {
				return nil, err
			}
			o.Watch = true
		case "c", "copy":
			if err := noValue(); err != nil {
				return nil, err
			}
			o.Copy = true
		case "i", "input":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			o.inputFlag = v
		case "o", "output":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			o.Output = v
			outputGiven = true
		case "t", "time":
			v, err := takeValue()
			if err != nil {
				return nil, err
			}
			sec, err := parseSeconds(v)
			if err != nil {
				return nil, fmt.Errorf("flag %s: %w", flagUsage(name, long), err)
			}
			o.Time = time.Duration(sec) * time.Second
			o.timeSet = true
		default:
			return nil, fmt.Errorf("unknown flag %s", flagUsage(name, long))
		}
		i++
	}

	// Resolve the parameters, applying the compiled-in defaults for anything
	// that was not set on the command line.
	if len(positional) > 1 {
		return nil, fmt.Errorf("unexpected positional argument %q (input directory may be given at most once)", positional[1])
	}

	if len(positional) == 1 {
		if o.inputFlag != "" && o.inputFlag != positional[0] {
			return nil, fmt.Errorf("input directory given both via -i/--input and as a positional argument with different values")
		}
		o.Input = positional[0]
	} else {
		o.Input = o.inputFlag
	}
	if o.Input == "" {
		o.Input = defaultInputDir
	}
	if !outputGiven {
		o.Output = o.Input
	}
	if !o.timeSet {
		o.Time = defaultTimeSeconds * time.Second
	}
	return o, nil
}

// usageText documents the options and the supported invocation styles.
func usageText() string {
	return `Usage:
  ipa-renamer [options] [INPUT]

Rename ipa file with bundle identifier read from the app's Info.plist inside the archive.

Arguments:
  INPUT                   Directory to scan for .ipa files
                          (default: current working directory)

Options:
  -i, --input <DIR>       Input directory; equivalent to INPUT. Errors when
                          both are given with different values.
  -o, --output <DIR>      Output directory (default: the input directory).
  -c, --copy              Copy the renamed file instead of renaming it, so the
                          source .ipa is kept.
  -t, --time <SECONDS>    In watch mode, process a .ipa only after it has been
                          unchanged for this many seconds (default: 5).
  -w, --watch             Watch INPUT and process new/changed .ipa files as
                          they appear, instead of scanning once and exiting.
  -h, --help              Show this help and exit.
  -V, --version           Show the version and exit.

Examples:
  ipa-renamer -i /path/to/input -o /path/to/output
  ipa-renamer -o /path/to/output
  ipa-renamer -c -i /path/to/input -o /path/to/output
  ipa-renamer -w -t 8 -i /path/to/input -o /path/to/output
`
}
