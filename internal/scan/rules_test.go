package scan

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
)

// fakeResolver implements Resolver over inline records, keyed by the bare anchor
// number callers pass to Lookup.
type fakeResolver struct {
	byNum map[string]catalog.Record
	all   []catalog.Record
}

func (f fakeResolver) Lookup(ref string) (catalog.Record, bool) {
	r, ok := f.byNum[ref]
	return r, ok
}

func (f fakeResolver) Search(q string) []catalog.Record {
	q = strings.ToLower(q)
	var out []catalog.Record
	for _, r := range f.all {
		if strings.Contains(strings.ToLower(r.Title), q) {
			out = append(out, r)
		}
	}
	return out
}

var reAnchor = regexp.MustCompile(`^\d{3,5}(-\d+)?$`)

func TestRulesWellFormed(t *testing.T) {
	for _, r := range Rules {
		if r.Category == "" {
			t.Errorf("rule for concern %q has no category", r.Concern)
		}
		if len(r.Anchors) == 0 {
			t.Errorf("rule %q has no anchors", r.Category)
		}
		for _, a := range r.Anchors {
			if !reAnchor.MatchString(a) {
				t.Errorf("rule %q anchor %q is not a bare ISO number", r.Category, a)
			}
		}
		if _, ok := rulesByConcern[r.Concern]; !ok {
			t.Errorf("concern %q missing from rulesByConcern index", r.Concern)
		}
	}
}

func TestCategoryForTerm(t *testing.T) {
	cases := map[string]string{
		"ai":                      "Artificial Intelligence",
		"AI":                      "Artificial Intelligence",
		"devops":                  "DevOps",
		"infosec":                 "Information Security",
		"Information Security":    "Information Security",
		"Artificial Intelligence": "Artificial Intelligence",
	}
	for term, want := range cases {
		got, ok := CategoryForTerm(term)
		if !ok || got != want {
			t.Errorf("CategoryForTerm(%q) = %q,%v; want %q", term, got, ok, want)
		}
	}
	// "security" is neither a concern key nor an exact category, so it must not
	// resolve here (it is left to fuzzy header matching).
	if _, ok := CategoryForTerm("security"); ok {
		t.Error(`CategoryForTerm("security") should not resolve exactly`)
	}
}

func TestResolveRuleAnchorsAndMissing(t *testing.T) {
	res := fakeResolver{byNum: map[string]catalog.Record{
		"27001": {Reference: "ISO/IEC 27001:2022", Title: "ISMS Requirements", StageCode: 6060},
		"27002": {Reference: "ISO/IEC 27002:2022", Title: "Information security controls", StageCode: 6060},
		// 27005 intentionally absent.
	}}
	rule := rulesByConcern[ConcernInfosec]
	recs, missing := resolveRule(rule, res, false, true)
	if len(recs) != 2 {
		t.Fatalf("want 2 resolved anchors, got %d", len(recs))
	}
	if len(missing) != 1 || missing[0] != "27005" {
		t.Errorf("want missing=[27005], got %v", missing)
	}
	for _, r := range recs {
		if r.discovered {
			t.Error("anchor records must not be flagged discovered")
		}
	}
}

func TestResolveRulePublishedOnly(t *testing.T) {
	res := fakeResolver{byNum: map[string]catalog.Record{
		"27001": {Reference: "ISO/IEC 27001:2022", Title: "ISMS", StageCode: 6060},   // published
		"27002": {Reference: "ISO/IEC 27002:DRAFT", Title: "draft", StageCode: 5000}, // approval-stage draft
		"27005": {Reference: "ISO/IEC 27005:2022", Title: "risk", StageCode: 9060},   // confirmed
	}}
	rule := rulesByConcern[ConcernInfosec]
	recs, _ := resolveRule(rule, res, false, true)
	for _, r := range recs {
		if r.rec.StageCode == 5000 {
			t.Error("publishedOnly should exclude draft stage 5000")
		}
	}
	if len(recs) != 2 {
		t.Errorf("want 2 published anchors (27001, 27005), got %d", len(recs))
	}
	// With drafts included, all three resolve.
	recs2, _ := resolveRule(rule, res, false, false)
	if len(recs2) != 3 {
		t.Errorf("want 3 with drafts included, got %d", len(recs2))
	}
}

func TestResolveRuleDiscoverDedupes(t *testing.T) {
	anchor := catalog.Record{Reference: "ISO/IEC 27001:2022", Title: "Information security ISMS", StageCode: 6060, ICS: []string{"35.030 IT Security"}}
	extra := catalog.Record{Reference: "ISO/IEC 27099:2022", Title: "Information security framework", StageCode: 6060, ICS: []string{"35.030 IT Security"}}
	res := fakeResolver{
		byNum: map[string]catalog.Record{"27001": anchor},
		all:   []catalog.Record{anchor, extra},
	}
	rule := Rule{Concern: ConcernInfosec, Category: "Information Security", Anchors: []string{"27001"}, DiscoverTerms: []string{"information security"}, DiscoverICS: "35.030"}
	recs, _ := resolveRule(rule, res, true, true)
	if len(recs) != 2 {
		t.Fatalf("want anchor + 1 discovered, got %d", len(recs))
	}
	var anchorCount, discoveredCount int
	for _, r := range recs {
		if r.rec.Reference == anchor.Reference && r.discovered {
			t.Error("anchor must win over discovery duplicate")
		}
		if r.discovered {
			discoveredCount++
		} else {
			anchorCount++
		}
	}
	if anchorCount != 1 || discoveredCount != 1 {
		t.Errorf("want 1 anchor + 1 discovered, got %d/%d", anchorCount, discoveredCount)
	}
}
