package catalog

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Catalog provides offline lookup/search over loaded Records.
type Catalog struct {
	records []Record
	byRef   map[string]int // normalized reference -> index
	lower   []searchFields // per-record lowercased fields, parallel to records
}

// searchFields holds the lowercased fields scanned by Search, precomputed once
// in New so each search does not re-lowercase the whole catalogue.
type searchFields struct {
	ref, title, scope string
}

// New builds a Catalog and its lookup/search indexes.
func New(recs []Record) *Catalog {
	c := &Catalog{
		records: recs,
		byRef:   make(map[string]int, len(recs)),
		lower:   make([]searchFields, len(recs)),
	}
	for i, r := range recs {
		c.byRef[normalize(r.Reference)] = i
		c.lower[i] = searchFields{
			ref:   strings.ToLower(r.Reference),
			title: strings.ToLower(r.Title),
			scope: strings.ToLower(r.Scope),
		}
	}
	return c
}

// normalize lowercases s and strips all whitespace (including non-breaking
// spaces and other Unicode space variants) so references compare equal
// regardless of the spacing used by the source data or the user's query.
func normalize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}

var reBareNum = regexp.MustCompile(`(\d{3,5})`)

// isPublished reports whether a stage code represents a published or confirmed
// standard (ISO harmonized stage groups 60 = Published and 90 = Review/Confirmed).
// Both groups represent currently effective standards, as opposed to drafts
// (stages 10–50) or withdrawn standards (stage 95).
func isPublished(code int) bool { g := code / 100; return g == 60 || g == 90 }

// isDerivative reports whether ref is an amendment, corrigendum, or similar
// derivative document (e.g. "/Amd 1:2024", "/Cor 1:2009").
func isDerivative(ref string) bool {
	lower := strings.ToLower(ref)
	return strings.Contains(lower, "/amd") || strings.Contains(lower, "/cor")
}

// numberMatches reports whether num is the standard-number component of ref.
// It requires that the digit run is not adjacent to another digit on either
// side, so "9001" does not match inside "29001".
func numberMatches(ref, num string) bool {
	idx := 0
	for {
		i := strings.Index(ref[idx:], num)
		if i < 0 {
			return false
		}
		pos := idx + i
		before := pos - 1
		after := pos + len(num)
		okBefore := before < 0 || !isDigit(ref[before])
		okAfter := after >= len(ref) || !isDigit(ref[after])
		if okBefore && okAfter {
			return true
		}
		idx = pos + 1
	}
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// Lookup resolves a reference exactly, then loosely. For a bare number it
// prefers the base standard over amendments/corrigenda, then the non-replaced
// (current) edition, then the most recent by reference string.
func (c *Catalog) Lookup(ref string) (Record, bool) {
	if i, ok := c.byRef[normalize(ref)]; ok {
		return c.records[i], true
	}
	m := reBareNum.FindString(ref)
	if m == "" {
		return Record{}, false
	}
	var matches []Record
	for _, r := range c.records {
		if numberMatches(r.Reference, m) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return Record{}, false
	}
	sort.SliceStable(matches, func(a, b int) bool {
		ra, rb := lookupRank(matches[a]), lookupRank(matches[b])
		if ra != rb {
			return ra < rb // lower rank sorts first
		}
		return matches[a].Reference > matches[b].Reference
	})
	return matches[0], true
}

// lookupRank orders bare-number candidates: base standards before
// amendments/corrigenda, published before drafts, current before replaced.
// Lower values sort first; each criterion is weighted so a more important
// criterion always dominates a less important one.
func lookupRank(r Record) int {
	rank := 0
	if isDerivative(r.Reference) {
		rank += 4
	}
	if !isPublished(r.StageCode) {
		rank += 2
	}
	if r.ReplacedBy != "" {
		rank += 1
	}
	return rank
}

// Search returns records matching all query tokens, ranked
// reference > title > scope.
func (c *Catalog) Search(query string) []Record {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return []Record{}
	}
	type scored struct {
		r     Record
		score int
	}
	var hits []scored
	for i, r := range c.records {
		f := c.lower[i]
		ok := true
		score := 0
		for _, tok := range tokens {
			switch {
			case strings.Contains(f.ref, tok):
				score += 3
			case strings.Contains(f.title, tok):
				score += 2
			case strings.Contains(f.scope, tok):
				score += 1
			default:
				ok = false
			}
		}
		if ok {
			hits = append(hits, scored{r, score})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score > hits[b].score
		}
		return hits[a].r.Reference < hits[b].r.Reference
	})
	out := make([]Record, len(hits))
	for i, h := range hits {
		out[i] = h.r
	}
	return out
}
