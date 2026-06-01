package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindByConvention(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ISO-IEC-27001-2022.pdf"), []byte("x"), 0o644)
	lib := New(dir, "")
	got, ok := lib.Find("ISO/IEC 27001:2022")
	if !ok {
		t.Fatal("convention match failed")
	}
	if filepath.Base(got) != "ISO-IEC-27001-2022.pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestFindByConventionBareNumber(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "iso27001.md"), []byte("x"), 0o644)
	lib := New(dir, "")
	if _, ok := lib.Find("27001"); !ok {
		t.Fatal("bare-number convention match failed")
	}
}

func TestIndexOverridesConvention(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ISO-IEC-27001-2022.pdf"), []byte("x"), 0o644)
	custom := filepath.Join(dir, "my-copy.txt")
	os.WriteFile(custom, []byte("x"), 0o644)
	idx := filepath.Join(dir, "index.yaml")
	os.WriteFile(idx, []byte("entries:\n  \"ISO/IEC 27001:2022\": my-copy.txt\n"), 0o644)
	lib := New(dir, idx)
	got, ok := lib.Find("ISO/IEC 27001:2022")
	if !ok || filepath.Base(got) != "my-copy.txt" {
		t.Fatalf("index override failed: %q ok=%v", got, ok)
	}
}
