package main

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// alreadyNamedPattern matches the naming rule <raw>@<bundle-id>.ipa.
var alreadyNamedPattern = regexp.MustCompile(`^[^@]+@[^@]+\.ipa$`)

func isAlreadyNamed(name string) bool {
	return alreadyNamedPattern.MatchString(name)
}

// isIPASource reports whether path looks like an .ipa file (case-insensitive).
func isIPASource(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".ipa")
}

// rawNameOf strips the directory and extension of an archive name,
// e.g. "/in/DumpApp_1.0.4.ipa" => "DumpApp_1.0.4".
func rawNameOf(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if name == "" {
		name = "unknown"
	}
	return name
}

// renamedFileName builds the target file name "<raw>@<bundle-id>.ipa".
func renamedFileName(raw, bundleID string) string {
	return raw + "@" + bundleID + ".ipa"
}

// copyFile streams src onto dst, overwriting dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	buf := make([]byte, 1<<20)
	_, cErr := io.CopyBuffer(out, in, buf)
	closeErr := out.Close()
	if cErr != nil {
		return cErr
	}
	return closeErr
}

// RenameOne copies a source .ipa into cfg.Output as <raw>@<bundle-id>.ipa,
// reading the bundle identifier from the archive's Info.plist. The source file
// is left untouched.
func RenameOne(cfg Config, srcPath string) (string, error) {
	bundleID, err := ipaBundleID(srcPath)
	if err != nil {
		return "", err
	}
	dst := filepath.Join(cfg.Output, renamedFileName(rawNameOf(srcPath), bundleID))
	if err := copyFile(srcPath, dst); err != nil {
		return "", err
	}
	return dst, nil
}

// Process inspects one candidate path and, when it is a plain .ipa file that is
// not already correctly named, renames it. It returns the destination path on a
// successful rename, "" when the path was skipped, or an error on failure.
func Process(cfg Config, path string) (string, error) {
	if !isIPASource(path) {
		return "", nil
	}
	if isAlreadyNamed(filepath.Base(path)) {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if info.IsDir() {
		return "", nil
	}
	return RenameOne(cfg, path)
}
