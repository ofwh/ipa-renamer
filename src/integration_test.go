package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestIPA creates a minimal but structurally valid .ipa containing a main
// app Info.plist with the given bundle id, plus a decoy entry.
func writeTestIPA(t *testing.T, path, bundleID string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>CFBundleIdentifier</key><string>` + bundleID + `</string></dict></plist>`

	entries := map[string]string{
		"Payload/Demo.app/Info.plist":               plist,
		"Payload/Demo.app/embedded.mobileprovision": "decoy",
		"Payload/Other.app/Info.plist":              plist, // must not be picked first
	}
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}

func TestRenameIPARoundtrip(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(in, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	ipa := filepath.Join(in, "DumpApp_1.0.4.ipa")
	writeTestIPA(t, ipa, "com.dumpapp")

	cfg := Config{Input: in, Output: out, Time: time.Second}

	dst, err := Process(cfg, ipa)
	if err != nil {
		t.Fatalf("Process(%s) error: %v", ipa, err)
	}
	want := filepath.Join(out, "DumpApp_1.0.4@com.dumpapp.ipa")
	if dst != want {
		t.Errorf("dst = %q, want %q", dst, want)
	}
	if info, err := os.Stat(want); err != nil || info.IsDir() {
		t.Errorf("renamed file missing at %s: %v", want, err)
	}

	// An already-correctly-named file is skipped (returns "").
	skip, err := Process(cfg, want)
	if err != nil {
		t.Fatalf("Process on already-named error: %v", err)
	}
	if skip != "" {
		t.Errorf("already-named file should be skipped, got dst %q", skip)
	}

	// The source archive is preserved (copy, not move).
	if _, err := os.Stat(ipa); err != nil {
		t.Errorf("source .ipa should be kept: %v", err)
	}
}

func TestRenameIPABadArchive(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(in, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	bad := filepath.Join(in, "broken.ipa")
	if err := os.WriteFile(bad, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Process(Config{Input: in, Output: out}, bad); err == nil {
		t.Error("corrupt .ipa should return an error")
	}

	// Non-ipa files and directories are ignored.
	cfg := Config{Input: in, Output: out}
	if dst, err := Process(cfg, filepath.Join(in, "notes.txt")); err != nil || dst != "" {
		t.Errorf("non-ipa file: dst=%q err=%v, want skip", dst, err)
	}
}
