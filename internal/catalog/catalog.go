package catalog

import (
	"regexp"
	"sort"
	"strings"
)

// Catalog provides offline lookup/search over loaded Records.
type Catalog struct {
	records []Record
	byRef   map[string]int // normalized reference -> index
}

// New builds a Catalog and its lookup index.
func New(recs []Record) *Catalog {
	c := &Catalog{records: recs, byRef: make(map[string]int, len(recs))}
	for i, r := range recs {
		c.byRef[normalize(r.Reference)] = i
	}
	return c
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return s
}

var reBareNum = regexp.MustCompile(`(\d{3,5})`)

// Lookup resolves a reference exactly, then loosely. For a bare number it
// prefers the non-replaced (current) edition.
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
		if strings.Contains(r.Reference, m) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return Record{}, false
	}
	sort.SliceStable(matches, func(a, b int) bool {
		ca, cb := matches[a].ReplacedBy == "", matches[b].ReplacedBy == ""
		if ca != cb {
			return ca
		}
		return matches[a].Reference > matches[b].Reference
	})
	return matches[0], true
}

// Search returns records matching all query tokens, ranked
// reference > title > scope.
func (c *Catalog) Search(query string) []Record {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return nil
	}
	type scored struct {
		r     Record
		score int
	}
	var hits []scored
	for _, r := range c.records {
		ref := strings.ToLower(r.Reference)
		title := strings.ToLower(r.Title)
		scope := strings.ToLower(r.Scope)
		ok := true
		score := 0
		for _, tok := range tokens {
			switch {
			case strings.Contains(ref, tok):
				score += 3
			case strings.Contains(title, tok):
				score += 2
			case strings.Contains(scope, tok):
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
