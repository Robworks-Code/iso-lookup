package parse

import "testing"

func TestParseMarkdown(t *testing.T) {
	doc, err := Parse("testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("no sections")
	}
	flat := doc.Flatten()
	var hasScope, has41 bool
	for _, s := range flat {
		if s.Number == "1" && s.Title == "Scope" {
			hasScope = true
		}
		if s.Number == "4.1" {
			has41 = true
		}
	}
	if !hasScope || !has41 {
		t.Fatalf("missing sections: scope=%v 4.1=%v (%+v)", hasScope, has41, flat)
	}
}

func TestParseText(t *testing.T) {
	doc, err := Parse("testdata/sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("expected sections from numbered text")
	}
}
