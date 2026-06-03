package catalog

import "testing"

func filterFixtures() []Record {
	return []Record{
		{Reference: "ISO/IEC 27001:2022", Status: "Published", StageCode: 6060, PublishedDate: "2022-10-25", Committee: "ISO/IEC JTC 1/SC 27 — Information security", ICS: []string{"35.030 IT Security"}},
		{Reference: "ISO/IEC 27001:2013", Status: "Withdrawn", StageCode: 9599, PublishedDate: "2013-10-01", Committee: "ISO/IEC JTC 1/SC 27 — Information security", ICS: []string{"35.030 IT Security"}},
		{Reference: "ISO 9001:2015", Status: "Review", StageCode: 9092, PublishedDate: "2015-09-15", Committee: "ISO/TC 176/SC 2 — Quality systems", ICS: []string{"03.100.70 Management systems"}},
		{Reference: "ISO/FDIS 12345", Status: "Approval", StageCode: 5000, PublishedDate: "", Committee: "ISO/TC 999", ICS: nil},
	}
}

func refsOf(recs []Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Reference
	}
	return out
}

func TestFilterEmptyMatchesAll(t *testing.T) {
	recs := filterFixtures()
	got := Filter{}.Apply(recs)
	if len(got) != len(recs) {
		t.Fatalf("empty filter should match all %d, got %d", len(recs), len(got))
	}
}

func TestFilterPublishedOnly(t *testing.T) {
	got := Filter{PublishedOnly: true}.Apply(filterFixtures())
	// 27001:2022 (60) and 9001:2015 (90) are effective; 27001:2013 (95) and FDIS (50) are not.
	if len(got) != 2 {
		t.Fatalf("expected 2 effective standards, got %d: %v", len(got), refsOf(got))
	}
}

func TestFilterStatusSubstring(t *testing.T) {
	got := Filter{Status: "withdrawn"}.Apply(filterFixtures())
	if len(got) != 1 || got[0].Reference != "ISO/IEC 27001:2013" {
		t.Fatalf("status filter failed: %v", refsOf(got))
	}
}

func TestFilterICSPrefix(t *testing.T) {
	got := Filter{ICS: "35.030"}.Apply(filterFixtures())
	if len(got) != 2 {
		t.Fatalf("expected 2 ICS 35.030 matches, got %d: %v", len(got), refsOf(got))
	}
	// Broader prefix matches the same plus nothing else here.
	if g := (Filter{ICS: "03"}).Apply(filterFixtures()); len(g) != 1 || g[0].Reference != "ISO 9001:2015" {
		t.Fatalf("ICS prefix '03' failed: %v", refsOf(g))
	}
}

func TestFilterCommittee(t *testing.T) {
	got := Filter{Committee: "sc 27"}.Apply(filterFixtures())
	if len(got) != 2 {
		t.Fatalf("committee filter failed: %v", refsOf(got))
	}
}

func TestFilterYear(t *testing.T) {
	got := Filter{Year: "2022"}.Apply(filterFixtures())
	if len(got) != 1 || got[0].Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("year filter failed: %v", refsOf(got))
	}
}

func TestFilterCombined(t *testing.T) {
	got := Filter{Committee: "sc 27", PublishedOnly: true}.Apply(filterFixtures())
	if len(got) != 1 || got[0].Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("combined filter failed: %v", refsOf(got))
	}
}

func TestFilterCommitteeBoundary(t *testing.T) {
	recs := []Record{
		{Reference: "A", Committee: "ISO/TC 17 — Steel"},
		{Reference: "B", Committee: "ISO/TC 176/SC 2 — Quality management"},
	}
	// "TC 17" must not match "TC 176" (trailing digit).
	if g := (Filter{Committee: "TC 17"}).Apply(recs); len(g) != 1 || g[0].Reference != "A" {
		t.Fatalf("'TC 17' should match only TC 17, not TC 176: %v", refsOf(g))
	}
	// Sub-committee substrings still work when the boundary is a non-digit.
	if g := (Filter{Committee: "SC 2"}).Apply(recs); len(g) != 1 || g[0].Reference != "B" {
		t.Fatalf("'SC 2' should match TC 176/SC 2: %v", refsOf(g))
	}
}

func TestFilterYearBoundary(t *testing.T) {
	recs := []Record{
		{Reference: "X", PublishedDate: "2022-10-25"},
		{Reference: "Y", PublishedDate: "2009-01-01"},
	}
	// Partial years must not match a longer year.
	if g := (Filter{Year: "20"}).Apply(recs); len(g) != 0 {
		t.Fatalf("partial year '20' should match nothing, got %v", refsOf(g))
	}
	if g := (Filter{Year: "202"}).Apply(recs); len(g) != 0 {
		t.Fatalf("partial year '202' should match nothing, got %v", refsOf(g))
	}
	if g := (Filter{Year: "2022"}).Apply(recs); len(g) != 1 || g[0].Reference != "X" {
		t.Fatalf("year '2022' failed: %v", refsOf(g))
	}
}

func TestSortByReference(t *testing.T) {
	recs := filterFixtures()
	SortBy(recs, "reference")
	if recs[0].Reference != "ISO 9001:2015" {
		t.Fatalf("sort by reference failed: %v", refsOf(recs))
	}
}

func TestSortByDateNewestFirst(t *testing.T) {
	recs := filterFixtures()
	SortBy(recs, "date")
	if recs[0].Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("sort by date should put newest first: %v", refsOf(recs))
	}
}

func TestValidSortKey(t *testing.T) {
	if !ValidSortKey("date") || ValidSortKey("bogus") {
		t.Fatal("ValidSortKey wrong")
	}
}
