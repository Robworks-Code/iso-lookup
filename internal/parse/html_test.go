package parse

import "testing"

func TestParseHTML(t *testing.T) {
	doc, err := Parse("testdata/sample.html")
	if err != nil {
		t.Fatal(err)
	}
	flat := doc.Flatten()
	var hasScope, has41 bool
	for _, s := range flat {
		if s.Number == "1" && s.Title == "Scope" {
			hasScope = true
			if s.Body == "" {
				t.Error("scope body empty")
			}
		}
		if s.Number == "4.1" {
			has41 = true
		}
	}
	if !hasScope || !has41 {
		t.Fatalf("missing sections (%+v)", flat)
	}
}
