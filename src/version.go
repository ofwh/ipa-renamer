package main

// version is reported by -V/--version. Release and container builds override it
// at link time with -ldflags "-X main.version=<v>".
var version = "0.1.0"
