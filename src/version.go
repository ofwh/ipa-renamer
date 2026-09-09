package main

// version is reported by -V/--version. Release and container builds override it
// at link time with -ldflags "-X github.com/ofwh/ipa-renamer/src.version=<v>".
var version = "0.1.0"
