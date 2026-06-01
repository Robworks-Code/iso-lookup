package tui

import (
	"testing"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/parse"
)

func TestNewFlattensSections(t *testing.T) {
	doc := parse.Document{Sections: []parse.Section{
		{Number: "4", Title: "Context", Children: []parse.Section{{Number: "4.1", Title: "Understanding"}}},
	}}
	m := New(catalog.Record{Reference: "ISO/IEC 27001:2022"}, doc)
	if len(m.sections) != 2 {
		t.Fatalf("expected 2 flattened sections, got %d", len(m.sections))
	}
	// View on empty doc should not panic
	empty := New(catalog.Record{}, parse.Document{})
	if got := empty.View(); got == "" {
		t.Fatal("empty View returned empty string")
	}
}
