package main

import "testing"

func TestOpenArgs(t *testing.T) {
	bin, args := openCommand("darwin", "https://example.com")
	if bin != "open" || len(args) != 1 || args[0] != "https://example.com" {
		t.Fatalf("darwin: %s %v", bin, args)
	}
	bin, _ = openCommand("linux", "https://example.com")
	if bin != "xdg-open" {
		t.Fatalf("linux bin %s", bin)
	}
	bin, args = openCommand("windows", "https://example.com")
	if bin != "rundll32" || args[0] != "url.dll,FileProtocolHandler" {
		t.Fatalf("windows: %s %v", bin, args)
	}
}
