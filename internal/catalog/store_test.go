package catalog

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadIndex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "catalog.gob")
	in := []Record{{Reference: "ISO/IEC 27001:2022", Title: "ISMS"}}
	if err := SaveIndex(p, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Reference != in[0].Reference {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestLoadMissingIndex(t *testing.T) {
	_, err := LoadIndex(filepath.Join(t.TempDir(), "nope.gob"))
	if err != ErrNoIndex {
		t.Fatalf("want ErrNoIndex, got %v", err)
	}
}
