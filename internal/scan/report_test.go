package scan

import (
	"testing"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
)

func sampleDetection() Detection {
	return Detection{
		Root: "/proj",
		Components: []Component{
			{Name: "Go", Category: CatLanguage, Confidence: High, Evidence: []string{"go.mod"}, Concerns: []Concern{ConcernSDLC, ConcernSWQuality}},
			{Name: "OpenAI SDK", Category: CatAI, Confidence: High, Evidence: []string{"requirements.txt"}, Concerns: []Concern{ConcernAI}},
		},
	}
}

func sampleResolver() fakeResolver {
	pub := func(ref, title string) catalog.Record {
		return catalog.Record{Reference: ref, Title: title, Status: "Published", StageCode: 6060}
	}
	by := map[string]catalog.Record{
		"12207": pub("ISO/IEC/IEEE 12207:2017", "Software life cycle processes"),
		"15288": pub("ISO/IEC/IEEE 15288:2023", "System life cycle processes"),
		"25010": pub("ISO/IEC 25010:2023", "Product quality model"),
		"25040": pub("ISO/IEC 25040:2024", "Quality evaluation framework"),
		"42001": pub("ISO/IEC 42001:2023", "AI management system"),
		"23894": pub("ISO/IEC 23894:2023", "AI risk management"),
		"23053": pub("ISO/IEC 23053:2022", "Framework for AI systems using ML"),
		"22989": pub("ISO/IEC 22989:2022", "AI concepts and terminology"),
	}
	return fakeResolver{byNum: by}
}

func headers(rep Report) []string {
	out := make([]string, len(rep.Groups))
	for i, g := range rep.Groups {
		out[i] = g.Header
	}
	return out
}

func TestBuildGroupByComponent(t *testing.T) {
	rep := Build(sampleDetection(), sampleResolver(), BuildOptions{})
	if rep.GroupBy != GroupByComponent {
		t.Errorf("default GroupBy = %q, want component", rep.GroupBy)
	}
	hs := headers(rep)
	if len(hs) != 2 || hs[0] != "Go" || hs[1] != "OpenAI SDK" {
		t.Fatalf("headers = %v, want [Go OpenAI SDK]", hs)
	}
	// Go raises sdlc + sw_quality → 4 standards.
	if n := len(rep.Groups[0].Recommendations); n != 4 {
		t.Errorf("Go group has %d standards, want 4", n)
	}
}

func TestBuildGroupByCategory(t *testing.T) {
	rep := Build(sampleDetection(), sampleResolver(), BuildOptions{GroupBy: GroupByCategory})
	want := map[string]bool{"Software Lifecycle": false, "Software Quality": false, "Artificial Intelligence": false}
	for _, h := range headers(rep) {
		if _, ok := want[h]; ok {
			want[h] = true
		}
	}
	for cat, seen := range want {
		if !seen {
			t.Errorf("missing category group %q (got %v)", cat, headers(rep))
		}
	}
}

func TestBuildFilterComponent(t *testing.T) {
	rep := Build(sampleDetection(), sampleResolver(), BuildOptions{Component: "openai"})
	if len(rep.Groups) != 1 || rep.Groups[0].Header != "OpenAI SDK" {
		t.Fatalf("component filter headers = %v, want [OpenAI SDK]", headers(rep))
	}
	for _, r := range rep.Groups[0].Recommendations {
		if !contains(r.Components, "OpenAI SDK") {
			t.Errorf("recommendation %s not driven by OpenAI SDK", r.Record.Reference)
		}
	}
}

func TestBuildFilterCategoryAcrossComponentGrouping(t *testing.T) {
	// --category must match a recommendation's domain category even when the
	// report is grouped by component (where the category is not the header).
	rep := Build(sampleDetection(), sampleResolver(), BuildOptions{GroupBy: GroupByComponent, Category: "Lifecycle"})
	if len(rep.Groups) == 0 {
		t.Fatal("category filter dropped everything; expected lifecycle standards under the Go component")
	}
	for _, g := range rep.Groups {
		for _, r := range g.Recommendations {
			if !contains(r.Categories, "Software Lifecycle") {
				t.Errorf("standard %s kept but not in Software Lifecycle (%v)", r.Record.Reference, r.Categories)
			}
		}
	}
}

func TestBuildLimitPerGroup(t *testing.T) {
	rep := Build(sampleDetection(), sampleResolver(), BuildOptions{LimitPerGroup: 1})
	for _, g := range rep.Groups {
		if len(g.Recommendations) != 1 {
			t.Errorf("group %q has %d recs, want 1 (limit)", g.Header, len(g.Recommendations))
		}
		if g.Total < 1 {
			t.Errorf("group %q Total not recorded before limit", g.Header)
		}
	}
}

func TestBuildIncludeDrafts(t *testing.T) {
	res := sampleResolver()
	res.byNum["42001"] = catalog.Record{Reference: "ISO/IEC 42001:DRAFT", Title: "AI mgmt draft", StageCode: 4000}
	def := Build(sampleDetection(), res, BuildOptions{GroupBy: GroupByCategory, Category: "Artificial"})
	withDrafts := Build(sampleDetection(), res, BuildOptions{GroupBy: GroupByCategory, Category: "Artificial", IncludeDrafts: true})
	countAI := func(rep Report) int {
		for _, g := range rep.Groups {
			if g.Header == "Artificial Intelligence" {
				return len(g.Recommendations)
			}
		}
		return 0
	}
	if countAI(withDrafts) <= countAI(def) {
		t.Errorf("include-drafts should surface more AI standards: default=%d drafts=%d", countAI(def), countAI(withDrafts))
	}
}

func TestBuildEmptyDetection(t *testing.T) {
	rep := Build(Detection{Root: "/x"}, sampleResolver(), BuildOptions{})
	if len(rep.Groups) != 0 || len(rep.Components) != 0 {
		t.Errorf("empty detection should yield no groups/components, got %d/%d", len(rep.Groups), len(rep.Components))
	}
}
