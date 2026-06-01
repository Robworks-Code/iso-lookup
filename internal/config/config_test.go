package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirRespectsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdgtest")
	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/xdgtest", "iso-lookup")
	if got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), "iso-cfg-missing"))
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DocsDir != "" {
		t.Fatalf("expected empty DocsDir default, got %q", c.DocsDir)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), "iso-cfg-rt"))
	in := Config{DocsDir: "/docs", IndexFile: "/docs/index.yaml", Pager: "less"}
	if err := Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: %+v vs %+v", out, in)
	}
}
