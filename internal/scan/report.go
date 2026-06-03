package scan

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
)

// GroupBy values for Report grouping.
const (
	GroupByComponent = "component"
	GroupByCategory  = "category"
	GroupByICS       = "ics"
)

// GroupByKeys lists the recognized --group-by values.
var GroupByKeys = []string{GroupByComponent, GroupByCategory, GroupByICS}

// ValidGroupBy reports whether key is a recognized grouping.
func ValidGroupBy(key string) bool {
	for _, k := range GroupByKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Recommendation is one ISO standard suggested for the scanned project, with the
// detected components and concerns that drove it.
type Recommendation struct {
	Record     catalog.Record
	Concerns   []Concern
	Categories []string // domain categories this standard falls under
	Components []string // component names that raised the driving concerns
	Evidence   []string // files behind those components
	Rationale  string
	Confidence Confidence
	Discovered bool // true if surfaced by --discover rather than a curated anchor
}

// MarshalJSON flattens the catalog record inline so consumers see standard
// fields alongside the scan annotations, with confidence as a label.
func (r Recommendation) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Reference  string    `json:"reference"`
		Title      string    `json:"title"`
		Status     string    `json:"status"`
		URL        string    `json:"url"`
		ICS        []string  `json:"ics,omitempty"`
		Concerns   []Concern `json:"concerns"`
		Categories []string  `json:"categories"`
		Components []string  `json:"components"`
		Evidence   []string  `json:"evidence"`
		Rationale  string    `json:"rationale"`
		Confidence string    `json:"confidence"`
		Discovered bool      `json:"discovered"`
	}{
		Reference:  r.Record.Reference,
		Title:      r.Record.Title,
		Status:     r.Record.Status,
		URL:        r.Record.URL,
		ICS:        r.Record.ICS,
		Concerns:   r.Concerns,
		Categories: r.Categories,
		Components: r.Components,
		Evidence:   r.Evidence,
		Rationale:  r.Rationale,
		Confidence: r.Confidence.String(),
		Discovered: r.Discovered,
	})
}

// Group is a set of recommendations under one header (a component, a domain
// category, or an ICS code, per Report.GroupBy).
type Group struct {
	Header          string           `json:"header"`
	Recommendations []Recommendation `json:"recommendations"`
	Missing         []string         `json:"missing,omitempty"` // anchor refs absent from the catalog
	Total           int              `json:"total"`             // count before any per-group limit
}

// Report is the full scan result: the detected stack plus grouped standards.
type Report struct {
	Root       string      `json:"root"`
	GroupBy    string      `json:"group_by"`
	Components []Component `json:"components"`
	Groups     []Group     `json:"groups"`
	Truncated  bool        `json:"truncated"`
}

// BuildOptions controls report construction.
type BuildOptions struct {
	Discover      bool
	IncludeDrafts bool
	GroupBy       string
	Sort          string
	Category      string // keep only groups/recs matching this (substring, case-insensitive)
	Component     string // keep only recs driven by a matching component
	LimitPerGroup int    // 0 = no limit
}

// accum aggregates contributions to a single standard across concerns.
type accum struct {
	rec        catalog.Record
	concerns   []Concern
	components []string
	categories []string
	evidence   []string
	rationale  string
	confidence Confidence
	anyAnchor  bool
}

// Build runs the mapping/resolution pipeline over a Detection and produces a
// grouped Report. It is pure given a Resolver, so it is testable without a real
// catalog.
func Build(det Detection, res Resolver, opts BuildOptions) Report {
	if opts.GroupBy == "" {
		opts.GroupBy = GroupByComponent
	}
	publishedOnly := !opts.IncludeDrafts

	// Map each concern to the components that raised it (preserving order).
	concernComps := map[Concern][]Component{}
	var concernOrder []Concern
	for _, c := range det.Components {
		for _, cn := range c.Concerns {
			if _, seen := concernComps[cn]; !seen {
				concernOrder = append(concernOrder, cn)
			}
			concernComps[cn] = append(concernComps[cn], c)
		}
	}

	accums := map[string]*accum{}
	var accOrder []string
	missingByCat := map[string][]string{}

	for _, cn := range concernOrder {
		rule, ok := rulesByConcern[cn]
		if !ok {
			continue
		}
		drivers := concernComps[cn]
		resolvedRecs, missing := resolveRule(rule, res, opts.Discover, publishedOnly)
		if len(missing) > 0 {
			missingByCat[rule.Category] = appendUnique(missingByCat[rule.Category], missing...)
		}
		for _, rr := range resolvedRecs {
			a, ok := accums[rr.rec.Reference]
			if !ok {
				a = &accum{rec: rr.rec}
				accums[rr.rec.Reference] = a
				accOrder = append(accOrder, rr.rec.Reference)
			}
			a.concerns = appendUniqueConcern(a.concerns, cn)
			a.categories = appendUnique(a.categories, rule.Category)
			if a.rationale == "" {
				a.rationale = rule.Rationale
			}
			driverConf := Low
			for _, d := range drivers {
				a.components = appendUnique(a.components, d.Name)
				a.evidence = appendUnique(a.evidence, d.Evidence...)
				if d.Confidence > driverConf {
					driverConf = d.Confidence
				}
			}
			if !rr.discovered {
				a.anyAnchor = true
			} else if driverConf > Low {
				driverConf = Low // discovery is inherently lower confidence
			}
			if driverConf > a.confidence {
				a.confidence = driverConf
			}
		}
	}

	rep := Report{
		Root:       det.Root,
		GroupBy:    opts.GroupBy,
		Components: det.Components,
		Truncated:  det.Truncated,
	}
	recOf := func(ref string) Recommendation {
		a := accums[ref]
		return Recommendation{
			Record:     a.rec,
			Concerns:   a.concerns,
			Categories: a.categories,
			Components: a.components,
			Evidence:   a.evidence,
			Rationale:  a.rationale,
			Confidence: a.confidence,
			Discovered: !a.anyAnchor,
		}
	}

	switch opts.GroupBy {
	case GroupByCategory:
		rep.Groups = groupByCategory(accOrder, accums, recOf, missingByCat)
	case GroupByICS:
		rep.Groups = groupByICS(accOrder, accums, recOf)
	default:
		rep.Groups = groupByComponent(det.Components, accOrder, accums, recOf)
	}

	applyFilters(&rep, opts)
	for i := range rep.Groups {
		g := &rep.Groups[i]
		sortRecommendations(g.Recommendations, opts.Sort)
		g.Total = len(g.Recommendations)
		if opts.LimitPerGroup > 0 && len(g.Recommendations) > opts.LimitPerGroup {
			g.Recommendations = g.Recommendations[:opts.LimitPerGroup]
		}
	}
	return rep
}

func groupByComponent(comps []Component, accOrder []string, accums map[string]*accum, recOf func(string) Recommendation) []Group {
	var groups []Group
	for _, c := range comps {
		var recs []Recommendation
		for _, ref := range accOrder {
			if contains(accums[ref].components, c.Name) {
				recs = append(recs, recOf(ref))
			}
		}
		if len(recs) > 0 {
			groups = append(groups, Group{Header: c.Name, Recommendations: recs})
		}
	}
	return groups
}

func groupByCategory(accOrder []string, accums map[string]*accum, recOf func(string) Recommendation, missingByCat map[string][]string) []Group {
	var order []string
	for _, r := range Rules {
		if !contains(order, r.Category) {
			order = append(order, r.Category)
		}
	}
	var groups []Group
	for _, cat := range order {
		var recs []Recommendation
		for _, ref := range accOrder {
			if contains(accums[ref].categories, cat) {
				recs = append(recs, recOf(ref))
			}
		}
		if len(recs) > 0 || len(missingByCat[cat]) > 0 {
			groups = append(groups, Group{Header: cat, Recommendations: recs, Missing: missingByCat[cat]})
		}
	}
	return groups
}

func groupByICS(accOrder []string, accums map[string]*accum, recOf func(string) Recommendation) []Group {
	buckets := map[string][]Recommendation{}
	var codeOrder []string
	for _, ref := range accOrder {
		codes := icsCodes(accums[ref].rec.ICS)
		if len(codes) == 0 {
			codes = []string{"Unclassified"}
		}
		for _, code := range codes {
			if _, ok := buckets[code]; !ok {
				codeOrder = append(codeOrder, code)
			}
			buckets[code] = append(buckets[code], recOf(ref))
		}
	}
	sort.Strings(codeOrder)
	var groups []Group
	for _, code := range codeOrder {
		groups = append(groups, Group{Header: code, Recommendations: buckets[code]})
	}
	return groups
}

// applyFilters drops recommendations failing the --category/--component filters,
// then prunes empty groups. --category matches a recommendation's domain
// categories (and the group header), so it works regardless of --group-by.
func applyFilters(rep *Report, opts BuildOptions) {
	if opts.Category == "" && opts.Component == "" {
		return
	}
	var kept []Group
	for _, g := range rep.Groups {
		var recs []Recommendation
		for _, r := range g.Recommendations {
			if opts.Component != "" && !matchesTerm(opts.Component, r.Components...) {
				continue
			}
			if opts.Category != "" && !matchesTerm(opts.Category, append([]string{g.Header}, r.Categories...)...) {
				continue
			}
			recs = append(recs, r)
		}
		if len(recs) > 0 {
			g.Recommendations = recs
			g.Missing = nil
			kept = append(kept, g)
		}
	}
	rep.Groups = kept
}

// sortRecommendations orders a group's recommendations. The default (empty or
// "relevance") puts curated anchors before discovered items, higher confidence
// first, then reference. Named keys reuse catalog sort semantics.
func sortRecommendations(recs []Recommendation, key string) {
	switch strings.ToLower(key) {
	case "reference":
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Record.Reference < recs[j].Record.Reference })
	case "date":
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Record.PublishedDate > recs[j].Record.PublishedDate })
	case "status":
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Record.Status < recs[j].Record.Status })
	default:
		sort.SliceStable(recs, func(i, j int) bool {
			if recs[i].Discovered != recs[j].Discovered {
				return !recs[i].Discovered
			}
			if recs[i].Confidence != recs[j].Confidence {
				return recs[i].Confidence > recs[j].Confidence
			}
			return recs[i].Record.Reference < recs[j].Record.Reference
		})
	}
}

// HeaderGroup returns the first group whose header matches term
// (case-insensitive substring), without the component/concern fallback.
func (rep Report) HeaderGroup(term string) (Group, bool) {
	for _, g := range rep.Groups {
		if matchesTerm(term, g.Header) {
			return g, true
		}
	}
	return Group{}, false
}

// FindGroup returns the first group whose header, or whose recommendations'
// components/concerns, match term (case-insensitive). Used by `scan why`.
func (rep Report) FindGroup(term string) (Group, bool) {
	if g, ok := rep.HeaderGroup(term); ok {
		return g, true
	}
	// Fall back to matching a driving component or concern within any group.
	for _, g := range rep.Groups {
		for _, r := range g.Recommendations {
			cands := append([]string{}, r.Components...)
			for _, cn := range r.Concerns {
				cands = append(cands, string(cn))
			}
			if matchesTerm(term, cands...) {
				return g, true
			}
		}
	}
	return Group{}, false
}

// icsCodes extracts the leading code token from each "CODE Description" ICS entry.
func icsCodes(ics []string) []string {
	var out []string
	for _, e := range ics {
		if code := strings.Fields(e); len(code) > 0 {
			out = appendUnique(out, code[0])
		}
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func appendUnique(s []string, vs ...string) []string {
	for _, v := range vs {
		if !contains(s, v) {
			s = append(s, v)
		}
	}
	return s
}

func appendUniqueConcern(s []Concern, v Concern) []Concern {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
