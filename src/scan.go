package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

// walkIPA calls fn for every .ipa file in root, and returns how many it visited.
// Subdirectories are descended into only when cfg.Recursive is set. Directory
// symlinks are not followed, so a link loop cannot make this endless.
//
// The output directory is never walked into when it lies below root: it holds
// the results of a previous run, not new input.
func walkIPA(cfg Config, root string, fn func(path string)) int {
	if !cfg.Recursive {
		entries, err := os.ReadDir(root)
		if err != nil {
			logError("read directory %s: %v", root, err)
			return 0
		}
		count := 0
		for _, entry := range entries {
			if entry.IsDir() || !isIPASource(entry.Name()) {
				continue
			}
			fn(filepath.Join(root, entry.Name()))
			count++
		}
		return count
	}

	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable entry is skipped, not fatal: the rest of the tree
			// is still worth processing.
			logWarn("skip %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			if path != root && cfg.Output != cfg.Input && path == cfg.Output {
				return filepath.SkipDir
			}
			return nil
		}
		if isIPASource(path) {
			fn(path)
			count++
		}
		return nil
	})
	if err != nil {
		logWarn("walk %s: %v", root, err)
	}
	return count
}
