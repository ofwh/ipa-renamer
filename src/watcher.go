package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// settleTick is the debounce sweep interval.
	settleTick = 1 * time.Second
	// initialScanDelay delays the first scan until the watcher is armed.
	initialScanDelay = 500 * time.Millisecond
)

// Watch watches Input for .ipa files. Files already present are processed
// shortly after startup; later each new, modified or renamed .ipa is processed
// once it has stayed unchanged for cfg.Time. Watch returns on SIGINT/SIGTERM or
// a fatal watcher error.
func Watch(cfg Config) error {
	if err := cfg.ensureDirs(true); err != nil {
		return err
	}
	if err := cfg.checkDirs(); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	// jobs queues candidate paths; a single worker keeps renames sequential.
	jobs := make(chan string, 32)
	go func() {
		for path := range jobs {
			dst, err := Process(cfg, path)
			switch {
			case err != nil:
				logRenameFailed(path, err)
			case dst != "":
				logRenamed(path, dst)
			}
		}
	}()

	// pending tracks the last event time per path.
	var (
		mu      sync.Mutex
		pending = make(map[string]time.Time)
	)

	recordEvent := func(path string) {
		mu.Lock()
		pending[path] = time.Now()
		mu.Unlock()
	}

	// Debounce sweep: enqueue the paths that have stayed quiet for at least
	// cfg.Time, then forget them.
	go func() {
		ticker := time.NewTicker(settleTick)
		defer ticker.Stop()
		for range ticker.C {
			var due []string
			mu.Lock()
			for path, last := range pending {
				if time.Since(last) >= cfg.Time {
					due = append(due, path)
					delete(pending, path)
				}
			}
			mu.Unlock()
			for _, path := range due {
				logDebug("settled after %s: %s", cfg.Time, path)
				jobs <- path
			}
		}
	}()

	// Initial scan: queue .ipa files that were present when watching started.
	go func() {
		time.Sleep(initialScanDelay)
		entries, err := os.ReadDir(cfg.Input)
		if err != nil {
			logError("scan input directory: %v", err)
			return
		}
		count := 0
		for _, entry := range entries {
			if entry.IsDir() || !isIPASource(entry.Name()) {
				continue
			}
			jobs <- filepath.Join(cfg.Input, entry.Name())
			count++
		}
		logInfo("queued %d existing .ipa file(s)", count)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	if err := watcher.Add(cfg.Input); err != nil {
		return fmt.Errorf("watch directory %s: %w", cfg.Input, err)
	}
	logInfo("watching %s for .ipa files (settle delay %s)", cfg.Input, cfg.Time)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			logDebug("event %s: %s", event.Op, event.Name)
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0 {
				recordEvent(event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logError("watcher error: %v", err)
		case sig := <-sigs:
			logInfo("received %s, shutting down", sig)
			return nil
		}
	}
}
