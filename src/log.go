package main

import (
	"fmt"
	"os"
	"time"
)

const logTimeLayout = "2006-01-02 15:04:05"

// logAt writes one record as "<time> [<LEVEL>] <message>".
func logAt(w *os.File, level, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(w, "%s [%s] %s\n", time.Now().Format(logTimeLayout), level, msg)
}

// Info goes to stdout; warnings and errors go to stderr.
func logInfo(format string, a ...any)  { logAt(os.Stdout, "INFO", format, a...) }
func logWarn(format string, a ...any)  { logAt(os.Stderr, "WARN", format, a...) }
func logError(format string, a ...any) { logAt(os.Stderr, "ERROR", format, a...) }

// logRenamed and logRenameFailed keep the per-file wording identical between
// one-shot and watch mode.
func logRenamed(src, dst string) {
	logInfo("✓ %s -> %s", src, dst)
}

func logRenameFailed(src string, err error) {
	logError("𐄂 %s: %v", src, err)
}
