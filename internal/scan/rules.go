package scan

import (
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
)

// Resolver is the subset of *catalog.Catalog that scan needs. Decoupling via an
// interface keeps the scan package testable without a populated gob index.
type Resolver interface {
	Lookup(ref string) (catalog.Record, bool)
	Search(query string) []catalog.Record
}

// Rule maps a Concern to a report category and the ISO standards relevant to it.
// Anchors are curated, high-confidence references resolved via Resolver.Lookup.
// DiscoverTerms/DiscoverICS broaden the set via Resolver.Search when --discover
// is set. Anchors carry bare numbers (e.g. "27001"); Lookup picks the
// authoritative current edition.
type Rule struct {
	Concern       Concern
	Category      string
	Anchors       []string
	DiscoverTerms []string
	DiscoverICS   string
	Rationale     string
}

// Rules is the curated knowledge base. ISO standards address domains, not
// specific products, so each row maps a detected concern to the standards that
// govern that domain. Adding coverage is a one-line append.
var Rules = []Rule{
	{
		Concern: ConcernInfosec, Category: "Information Security",
		Anchors:       []string{"27001", "27002", "27005"},
		DiscoverTerms: []string{"information security", "cryptography"},
		DiscoverICS:   "35.030",
		Rationale:     "Authentication, tokens, or secret handling put this code in scope of an information security management system.",
	},
	{
		Concern: ConcernCloud, Category: "Cloud Security & Services",
		Anchors:       []string{"27017", "27018", "22123"},
		DiscoverTerms: []string{"cloud computing"},
		DiscoverICS:   "35.210",
		Rationale:     "Cloud SDKs or infrastructure code mean cloud security and service-model standards apply.",
	},
	{
		Concern: ConcernPrivacy, Category: "Privacy & PII",
		Anchors:       []string{"27701", "29100"},
		DiscoverTerms: []string{"privacy", "personally identifiable"},
		DiscoverICS:   "35.030",
		Rationale:     "User identity or payment handling implies processing of personal data, governed by privacy management standards.",
	},
	{
		Concern: ConcernSDLC, Category: "Software Lifecycle",
		Anchors:       []string{"12207", "15288"},
		DiscoverTerms: []string{"software life cycle"},
		DiscoverICS:   "35.080",
		Rationale:     "Any source project runs lifecycle processes (requirements, design, maintenance) covered by these standards.",
	},
	{
		Concern: ConcernSWQuality, Category: "Software Quality",
		Anchors:       []string{"25010", "25040"},
		DiscoverTerms: []string{"software quality"},
		DiscoverICS:   "35.080",
		Rationale:     "Product quality characteristics and evaluation criteria apply to the codebase.",
	},
	{
		Concern: ConcernTesting, Category: "Software Testing",
		Anchors:       []string{"29119"},
		DiscoverTerms: []string{"software testing"},
		DiscoverICS:   "35.080",
		Rationale:     "Test suites and CI test stages map to the software testing standard series.",
	},
	{
		Concern: ConcernITSM, Category: "IT Service Management",
		Anchors:       []string{"20000"},
		DiscoverTerms: []string{"service management"},
		DiscoverICS:   "35.080",
		Rationale:     "Observability and operated-service signals indicate IT service management practices.",
	},
	{
		Concern: ConcernAI, Category: "Artificial Intelligence",
		Anchors:       []string{"42001", "23894", "23053", "22989"},
		DiscoverTerms: []string{"artificial intelligence"},
		DiscoverICS:   "35.020",
		Rationale:     "Machine learning or large-language-model dependencies bring AI management and risk standards into scope.",
	},
	{
		Concern: ConcernQMS, Category: "Quality Management",
		Anchors:       []string{"9001"},
		DiscoverTerms: []string{"quality management systems"},
		DiscoverICS:   "03.120",
		Rationale:     "Organization-level quality management baseline for any product team.",
	},
	{
		Concern: ConcernDevOps, Category: "DevOps",
		Anchors:       []string{"32675", "12207"},
		DiscoverTerms: []string{"devops", "continuous"},
		DiscoverICS:   "35.080",
		Rationale:     "CI/CD pipelines map to DevOps and continuous delivery process standards.",
	},
	{
		Concern: ConcernAccessibility, Category: "Accessibility",
		Anchors:       []string{"9241-171", "40500"},
		DiscoverTerms: []string{"accessibility"},
		DiscoverICS:   "35.180",
		Rationale:     "A web or app UI carries accessibility obligations covered by these standards.",
	},
	{
		Concern: ConcernData, Category: "Data Management",
		Anchors:       []string{"11179", "27001"},
		DiscoverTerms: []string{"data management", "metadata registries"},
		DiscoverICS:   "35.040",
		Rationale:     "Database drivers imply data governance, metadata, and protection responsibilities.",
	},
	{
		Concern: ConcernCICD, Category: "Continuous Integration & Delivery",
		Anchors:       []string{"32675", "12207"},
		DiscoverTerms: []string{"continuous"},
		DiscoverICS:   "35.080",
		Rationale:     "Build/test/deploy pipelines map to DevOps and lifecycle process standards.",
	},
	{
		Concern: ConcernIaC, Category: "Configuration & Infrastructure",
		Anchors:       []string{"12207", "15288"},
		DiscoverTerms: []string{"configuration management"},
		DiscoverICS:   "35.080",
		Rationale:     "Infrastructure-as-code is managed configuration, covered by lifecycle and configuration management processes.",
	},
	{
		Concern: ConcernContainers, Category: "Containerization",
		Anchors:       []string{"19770"},
		DiscoverTerms: []string{"virtualization"},
		DiscoverICS:   "35.080",
		Rationale:     "Containers relate to software asset management and deployment; note ISO coverage of container tech specifically is thin.",
	},
	{
		Concern: ConcernWeb, Category: "Web & API",
		Anchors:       []string{"25010", "40500"},
		DiscoverTerms: []string{"web content"},
		DiscoverICS:   "35.080",
		Rationale:     "A web or API surface carries product-quality and accessibility obligations.",
	},
}

// CategoryForTerm resolves a term to a canonical domain category when it names a
// concern (e.g. "ai", "devops") or matches a category exactly (case-insensitive).
// It lets `scan why` target the right category before falling back to fuzzy
// substring matching, avoiding traps like "ai" matching "Containerization".
func CategoryForTerm(term string) (string, bool) {
	term = strings.TrimSpace(term)
	for _, r := range Rules {
		if strings.EqualFold(term, string(r.Concern)) || strings.EqualFold(term, r.Category) {
			return r.Category, true
		}
	}
	return "", false
}

// rulesByConcern indexes Rules for O(1) lookup during report building.
var rulesByConcern = func() map[Concern]Rule {
	m := make(map[Concern]Rule, len(Rules))
	for _, r := range Rules {
		m[r.Concern] = r
	}
	return m
}()

// resolved is one standard produced by a rule: the catalog record plus whether
// it came from curated anchors or from discovery.
type resolved struct {
	rec        catalog.Record
	discovered bool
}

// resolveRule turns a rule into catalog records. Anchors resolve via Lookup
// (skipping any not present, recorded in missing). With discover set, each
// DiscoverTerm is searched, scoped by DiscoverICS, and appended; discovered
// records duplicating an anchor are dropped. publishedOnly filters out drafts
// and withdrawn standards. The per-rule discovery count is capped.
func resolveRule(r Rule, res Resolver, discover, publishedOnly bool) (recs []resolved, missing []string) {
	have := map[string]bool{}
	for _, anchor := range r.Anchors {
		rec, ok := res.Lookup(anchor)
		if !ok {
			missing = append(missing, anchor)
			continue
		}
		if publishedOnly && !isPublishedRecord(rec) {
			continue
		}
		if have[rec.Reference] {
			continue
		}
		have[rec.Reference] = true
		recs = append(recs, resolved{rec: rec})
	}
	if !discover {
		return recs, missing
	}
	const maxDiscoverPerRule = 8
	added := 0
	for _, term := range r.DiscoverTerms {
		hits := res.Search(term)
		hits = catalog.Filter{ICS: r.DiscoverICS, PublishedOnly: publishedOnly}.Apply(hits)
		for _, rec := range hits {
			if added >= maxDiscoverPerRule {
				break
			}
			if have[rec.Reference] {
				continue
			}
			have[rec.Reference] = true
			recs = append(recs, resolved{rec: rec, discovered: true})
			added++
		}
	}
	return recs, missing
}

// isPublishedRecord reports whether a record is a currently-effective standard.
// It mirrors catalog's PublishedOnly filter without exposing internal stage
// logic: stage groups 60 (Published) and 90 (Review/Confirmed).
func isPublishedRecord(r catalog.Record) bool {
	g := r.StageCode / 100
	return g == 60 || g == 90
}
