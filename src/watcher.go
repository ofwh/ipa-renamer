package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// settleTick is the debounce sweep interval.
const settleTick = 1 * time.Second

// pending tracks the candidate paths seen so far. waiting holds the ones whose
// last event is too recent to act on; ready queues the ones that have stayed
// quiet long enough, oldest first.
//
// A path holds at most one valid entry in ready. A later event does not remove
// the entry — that would mean scanning the queue on every file system event —
// it invalidates it in place, and the worker drops invalidated entries as it
// pops them. That is what keeps a file which starts changing again from being
// processed halfway through its next copy.
type pending struct {
	mu      sync.Mutex
	waiting map[string]time.Time // path -> time of its last event
	ready   []string             // settled paths, oldest first
	queued  map[string]bool      // path -> holds a valid entry in ready
}

func newPending() *pending {
	return &pending{
		waiting: make(map[string]time.Time),
		queued:  make(map[string]bool),
	}
}

// touch records an event for path and restarts its settle delay. A path that
// was queued already has to stay quiet for another full delay before it is
// processed, so its queued entry is invalidated.
func (p *pending) touch(path string) {
	p.mu.Lock()
	p.queued[path] = false
	p.waiting[path] = time.Now()
	p.mu.Unlock()
}

// settle queues every path that has been quiet for at least quiet and returns
// the paths it queued.
func (p *pending) settle(quiet time.Duration) []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var queued []string
	now := time.Now()
	for path, last := range p.waiting {
		if now.Sub(last) < quiet || p.queued[path] {
			continue
		}
		delete(p.waiting, path)
		p.queued[path] = true
		p.ready = append(p.ready, path)
		queued = append(queued, path)
	}
	return queued
}

// pop returns the oldest path that is still queued, skipping the entries that a
// later event invalidated.
func (p *pending) pop() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.ready) > 0 {
		path := p.ready[0]
		p.ready = p.ready[1:]
		if !p.queued[path] {
			continue
		}
		p.queued[path] = false
		return path, true
	}
	return "", false
}

// waitingCount reports how many paths have not settled yet.
func (p *pending) waitingCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.waiting)
}

// watchDirs registers dir with the watcher and, when recursing, every
// directory below it. The watcher does not recurse on its own.
func watchDirs(w *fsnotify.Watcher, cfg Config, dir string) error {
	if err := w.Add(dir); err != nil {
		return err
	}
	if !cfg.Recursive {
		return nil
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == dir {
			return nil
		}
		if err := w.Add(path); err != nil {
			logWarn("watch directory %s: %v", path, err)
		}
		return nil
	})
	if err != nil {
		logWarn("scan subdirectories of %s: %v", dir, err)
	}
	return nil
}

// Watch watches Input for .ipa files and returns on SIGINT/SIGTERM or a fatal
// watcher error. The files already present at startup, and later each new,
// modified or renamed .ipa, are registered alike: a path is processed only once
// it has stayed unchanged for cfg.Time, so a copy that is still running when it
// is first noticed is left alone until it is finished. Subdirectories are
// included when cfg.Recursive is set. Files that are still settling are not
// processed on shutdown, but they are reported.
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

	pend := newPending()

	// wake nudges the worker after a sweep queued something; stop ends the
	// sweep and then the worker, which works off what is left in the queue
	// before it returns.
	wake := make(chan struct{}, 1)
	stop := make(chan struct{})
	sweepDone := make(chan struct{})
	workerDone := make(chan struct{})

	// The single worker keeps renames sequential.
	go func() {
		defer close(workerDone)
		process := func(path string) {
			dst, err := Process(cfg, path)
			switch {
			case err != nil:
				logRenameFailed(path, err)
			case dst != "":
				logRenamed(path, dst)
			}
		}
		for {
			path, ok := pend.pop()
			if ok {
				process(path)
				continue
			}
			select {
			case <-wake:
			case <-stop:
				// Process what the shutdown left queued, then return.
				for {
					path, ok := pend.pop()
					if !ok {
						return
					}
					process(path)
				}
			}
		}
	}()

	// Sweep: queue the paths that have stayed quiet for at least cfg.Time.
	go func() {
		defer close(sweepDone)
		ticker := time.NewTicker(settleTick)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				queued := pend.settle(cfg.Time)
				if len(queued) == 0 {
					continue
				}
				for _, path := range queued {
					logDebug("settled after %s: %s", cfg.Time, path)
				}
				select {
				case wake <- struct{}{}:
				default: // the worker is awake already
				}
			case <-stop:
				return
			}
		}
	}()

	// Stop the sweep before the worker, so that nothing is queued while the
	// worker drains.
	shutdown := func() {
		close(stop)
		<-sweepDone
		<-workerDone
	}

	// Watch first, then register what is already there: between the two, a file
	// that appears produces an event rather than being missed. Anything present
	// at this point settles for cfg.Time exactly like a later change would, so
	// a copy that is still running at startup is left alone until it is done.
	if err := watchDirs(watcher, cfg, cfg.Input); err != nil {
		shutdown()
		return fmt.Errorf("watch directory %s: %w", cfg.Input, err)
	}
	count := walkIPA(cfg, cfg.Input, pend.touch)
	logDebug("registered %d existing .ipa file(s), settling for %s", count, cfg.Time)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	scope := "input directory"
	if cfg.Recursive {
		scope = "input directory tree"
	}
	logInfo("watching %s %s for .ipa files (settle delay %s)", scope, cfg.Input, cfg.Time)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				shutdown()
				return nil
			}
			logDebug("event %s: %s", event.Op, event.Name)
			switch {
			case event.Op&fsnotify.Create != 0:
				// A directory that appears — created, or moved in with its
				// contents already in place — needs watching of its own, and
				// the files it brought along emit no events, so they are
				// registered by hand.
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if cfg.Recursive {
						if err := watchDirs(watcher, cfg, event.Name); err != nil {
							logWarn("watch directory %s: %v", event.Name, err)
						}
						walkIPA(cfg, event.Name, pend.touch)
					}
					continue
				}
				pend.touch(event.Name)
			case event.Op&fsnotify.Write != 0:
				pend.touch(event.Name)
			case event.Op&fsnotify.Rename != 0:
				// A Rename is also emitted for the path a file was moved away
				// from, which is every file this tool renames. Register it only
				// if something is still there to settle.
				if _, err := os.Stat(event.Name); err == nil {
					pend.touch(event.Name)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				shutdown()
				return nil
			}
			logError("watcher error: %v", err)
		case sig := <-sigs:
			logDebug("received %s, shutting down", sig)
			shutdown()
			if n := pend.waitingCount(); n > 0 {
				logInfo("%d path(s) still settling, left unprocessed", n)
			}
			return nil
		}
	}
}
