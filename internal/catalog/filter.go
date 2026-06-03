package catalog

import (
	"sort"
	"strings"
)

// Filter narrows a result set by metadata criteria. Zero-value fields are
// ignored, so an empty Filter matches every record.
type Filter struct {
	ICS           string // matches if any ICS entry's code begins with this prefix (e.g. "35.030" or "35")
	Committee     string // case-insensitive substring of the owning committee
	Status        string // case-insensitive substring of the status label (e.g. "published", "withdrawn", "review")
	Year          string // publication year; matches the YYYY prefix of the publication date
	PublishedOnly bool   // keep only currently-effective standards (stage groups 60 Published and 90 Review/Confirmed)
}

// Empty reports whether the filter has no active criteria.
func (f Filter) Empty() bool {
	return f.ICS == "" && f.Committee == "" && f.Status == "" && f.Year == "" && !f.PublishedOnly
}

func (f Filter) match(r Record) bool {
	if f.PublishedOnly && !isPublished(r.StageCode) {
		return false
	}
	if f.Status != "" && !strings.Contains(strings.ToLower(r.Status), strings.ToLower(f.Status)) {
		return false
	}
	if f.Committee != "" && !strings.Contains(strings.ToLower(r.Committee), strings.ToLower(f.Committee)) {
		return false
	}
	if f.Year != "" && !strings.HasPrefix(r.PublishedDate, f.Year) {
		return false
	}
	if f.ICS != "" {
		matched := false
		for _, e := range r.ICS {
			if strings.HasPrefix(e, f.ICS) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// Apply returns the records matching the filter, preserving their order.
// It always returns a non-nil slice.
func (f Filter) Apply(recs []Record) []Record {
	out := make([]Record, 0, len(recs))
	if f.Empty() {
		return append(out, recs...)
	}
	for _, r := range recs {
		if f.match(r) {
			out = append(out, r)
		}
	}
	return out
}

// SortKeys lists the recognized values for SortBy (besides "relevance").
var SortKeys = []string{"relevance", "reference", "date", "status"}

// SortBy reorders recs in place by the given key. "reference" sorts
// ascending, "date" newest-first, "status" alphabetical. "relevance" (or any
// unrecognized key) leaves the existing order — typically Search's ranking —
// untouched.
func SortBy(recs []Record, key string) {
	switch strings.ToLower(key) {
	case "reference", "ref":
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Reference < recs[j].Reference })
	case "date", "published":
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].PublishedDate > recs[j].PublishedDate })
	case "status":
		sort.SliceStable(recs, func(i, j int) bool { return recs[i].Status < recs[j].Status })
	}
}

// ValidSortKey reports whether key is a recognized SortBy value.
func ValidSortKey(key string) bool {
	for _, k := range SortKeys {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}
