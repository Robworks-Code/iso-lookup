package segment

import "testing"

func TestSectionsNumberedHeadings(t *testing.T) {
	raw := "Foreword\n\nIntro text.\n\n1 Scope\n\nThis clause.\n\n4 Context\n\n4.1 Understanding\n\nDetails.\n\nAnnex A\n\nA.5 Controls\n\nMore."
	secs := Sections(raw)
	if len(secs) == 0 {
		t.Fatal("no sections")
	}
	var four *Section
	for i := range secs {
		if secs[i].Number == "4" {
			four = &secs[i]
		}
	}
	if four == nil {
		t.Fatal("clause 4 not found")
	}
	if len(four.Children) != 1 || four.Children[0].Number != "4.1" {
		t.Fatalf("expected 4.1 child, got %+v", four.Children)
	}
}

func TestSectionsNoStructureFallback(t *testing.T) {
	secs := Sections("just a blob of text with no headings at all")
	if len(secs) != 1 || secs[0].Number != "" {
		t.Fatalf("expected single fallback section, got %+v", secs)
	}
}

func TestSectionsNormativeReferences(t *testing.T) {
	raw := "Normative references\n\nISO 9000, Quality management systems — Fundamentals and vocabulary.\n\n1 Scope\n\nThis standard applies."
	secs := Sections(raw)
	var hasNorm bool
	for _, s := range secs {
		if s.Title == "Normative references" {
			hasNorm = true
		}
	}
	if !hasNorm {
		t.Fatalf("Normative references section not detected (%+v)", secs)
	}
}
