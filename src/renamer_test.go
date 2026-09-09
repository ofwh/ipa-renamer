package main

import "testing"

func TestIsAlreadyNamed(t *testing.T) {
	already := []string{
		"DumpApp_1.0.4@com.dumpapp.ipa",
		"aszs@rn.notes.best.ipa",
	}
	for _, name := range already {
		if !isAlreadyNamed(name) {
			t.Errorf("isAlreadyNamed(%q) = false, want true", name)
		}
	}
	notYet := []string{
		"DumpApp_1.0.4.ipa",
		"a@b@c.ipa", // multiple @
		"no-bundle-id.ipa",
		"dir/aszs.ipa",
	}
	for _, name := range notYet {
		if isAlreadyNamed(name) {
			t.Errorf("isAlreadyNamed(%q) = true, want false", name)
		}
	}
}

func TestIsIPASource(t *testing.T) {
	if !isIPASource("/x/y/App.ipa") {
		t.Error("isIPASource should accept .ipa")
	}
	if !isIPASource("/x/y/App.IPA") {
		t.Error("isIPASource should be case-insensitive")
	}
	if isIPASource("/x/y/App.zip") || isIPASource("/x/y/App") {
		t.Error("isIPASource should reject non-.ipa paths")
	}
}

func TestRawNameOf(t *testing.T) {
	cases := map[string]string{
		"/in/DumpApp_1.0.4.ipa": "DumpApp_1.0.4",
		"aszs.ipa":              "aszs",
		"/x/.ipa":               "unknown",
	}
	for in, want := range cases {
		if got := rawNameOf(in); got != want {
			t.Errorf("rawNameOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenamedFileName(t *testing.T) {
	got := renamedFileName("DumpApp_1.0.4", "com.dumpapp")
	want := "DumpApp_1.0.4@com.dumpapp.ipa"
	if got != want {
		t.Errorf("renamedFileName = %q, want %q", got, want)
	}
}

func TestMatchInfoPlist(t *testing.T) {
	ok := []string{
		"Payload/Demo.app/Info.plist",
		"Payload/My App.app/Info.plist",
	}
	bad := []string{
		"Payload/Demo.app/nested/Info.plist",
		"Payload/Demo.app/Info.plist/extra",
		"Info.plist",
		"Payload/Demo.app/embedded.mobileprovision",
	}
	for _, name := range ok {
		if !matchInfoPlist(name) {
			t.Errorf("matchInfoPlist(%q) = false, want true", name)
		}
	}
	for _, name := range bad {
		if matchInfoPlist(name) {
			t.Errorf("matchInfoPlist(%q) = true, want false", name)
		}
	}
}
