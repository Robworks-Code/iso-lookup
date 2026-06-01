package parse

import (
	"os"
	"testing"
)

func TestParsePDFGraceful(t *testing.T) {
	const p = "testdata/sample.pdf"
	if _, err := os.Stat(p); err != nil {
		t.Skip("no sample.pdf fixture; provide one to exercise PDF extraction")
	}
	doc, err := Parse(p)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("expected at least a fallback section")
	}
}
