package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"

	plist "howett.net/plist"
)

// matchInfoPlist reports whether an archive entry is the main app Info.plist,
// i.e. a name shaped like "Payload/<App>.app/Info.plist".
func matchInfoPlist(name string) bool {
	return strings.Count(name, "/") == 2 && strings.HasSuffix(name, "/Info.plist")
}

// decodeBundleID parses an XML/binary plist and returns its CFBundleIdentifier.
func decodeBundleID(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	var info map[string]interface{}
	if err := plist.NewDecoder(bytes.NewReader(data)).Decode(&info); err != nil {
		return "", fmt.Errorf("decode Info.plist: %w", err)
	}
	id, ok := info["CFBundleIdentifier"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("CFBundleIdentifier missing or empty in Info.plist")
	}
	return id, nil
}

// ipaBundleID reads the CFBundleIdentifier of an .ipa archive by locating its
// main Payload/<App>.app/Info.plist entry and decoding it in memory.
func ipaBundleID(ipaPath string) (string, error) {
	r, err := zip.OpenReader(ipaPath)
	if err != nil {
		return "", fmt.Errorf("open ipa: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() || !matchInfoPlist(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("open %s: %w", f.Name, err)
		}
		id, err := decodeBundleID(rc)
		rc.Close()
		if err != nil {
			return "", fmt.Errorf("read %s: %w", f.Name, err)
		}
		return id, nil
	}
	return "", fmt.Errorf("no Payload/<App>.app/Info.plist entry found in %s", ipaPath)
}
