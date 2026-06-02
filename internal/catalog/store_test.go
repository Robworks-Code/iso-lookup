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

func TestSaveIndexAtomicReplace(t *testing.T) {
	p := filepath.Join(t.TempDir(), "catalog.gob")
	first := []Record{{Reference: "ISO 9001:2015", Title: "Quality"}}
	if err := SaveIndex(p, first); err != nil {
		t.Fatal(err)
	}
	second := []Record{
		{Reference: "ISO/IEC 27001:2022", Title: "ISMS"},
		{Reference: "ISO 9001:2015", Title: "Quality"},
	}
	if err := SaveIndex(p, second); err != nil {
		t.Fatal(err)
	}
	out, err := LoadIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 records after replace, got %d", len(out))
	}
	if out[0].Reference != second[0].Reference {
		t.Fatalf("expected %q, got %q", second[0].Reference, out[0].Reference)
	}
}
