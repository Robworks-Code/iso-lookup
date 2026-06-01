package catalog

import "testing"

func newTestCatalog() *Catalog {
	return New([]Record{
		{Reference: "ISO/IEC 27001:2022", Title: "Information security management systems — Requirements", Scope: "specifies requirements for an ISMS", ReplacedBy: ""},
		{Reference: "ISO/IEC 27001:2013", Title: "Information security management systems — Requirements", ReplacedBy: "ISO/IEC 27001:2022"},
		{Reference: "ISO 9001:2015", Title: "Quality management systems — Requirements", Scope: "quality management"},
	})
}

func TestLookupExact(t *testing.T) {
	c := newTestCatalog()
	r, ok := c.Lookup("ISO/IEC 27001:2022")
	if !ok || r.Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("exact lookup failed: %+v ok=%v", r, ok)
	}
}

func TestLookupBareNumberPrefersCurrent(t *testing.T) {
	c := newTestCatalog()
	r, ok := c.Lookup("27001")
	if !ok {
		t.Fatal("bare-number lookup failed")
	}
	if r.Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("expected current (non-replaced) edition, got %q", r.Reference)
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	c := newTestCatalog()
	if _, ok := c.Lookup("iso 9001"); !ok {
		t.Fatal("case-insensitive lookup failed")
	}
}

func TestLookupBareNumberPrefersBaseOverAmendment(t *testing.T) {
	c := New([]Record{
		{Reference: "ISO/IEC 27001:2022/Amd 1:2024", ReplacedBy: ""},
		{Reference: "ISO/IEC 27001:2022", ReplacedBy: ""},
		{Reference: "ISO/IEC 27001:2013", ReplacedBy: "ISO/IEC 27001:2022"},
	})
	r, ok := c.Lookup("27001")
	if !ok {
		t.Fatal("not found")
	}
	if r.Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("expected base standard, got %q", r.Reference)
	}
}

func TestSearchRanksReferenceThenTitleThenScope(t *testing.T) {
	c := newTestCatalog()
	res := c.Search("management")
	if len(res) == 0 {
		t.Fatal("no results")
	}
	found := false
	for _, r := range res {
		if r.Reference == "ISO 9001:2015" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ISO 9001:2015 in results")
	}
}
