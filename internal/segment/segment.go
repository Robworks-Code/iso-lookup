package segment

import (
	"regexp"
	"strings"

	"github.com/ringo380/iso-lookup/internal/docmodel"
)

// Section is an alias for the shared document section type.
type Section = docmodel.Section

var (
	reNumbered = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*|[A-Z]\.[0-9]+(?:\.[0-9]+)*)\s+(.+)$`)
	reNamed    = regexp.MustCompile(`^(Annex\s+[A-Z]|Foreword|Introduction|Scope|Normative references|Bibliography|Terms and definitions)\s*$`)
)

// Sections splits raw text into nested sections by heading heuristics.
// If no headings are found, returns a single section with the whole text.
func Sections(raw string) []Section {
	lines := strings.Split(raw, "\n")
	var flat []Section
	cur := -1
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if m := reNumbered.FindStringSubmatch(t); m != nil {
			flat = append(flat, Section{Number: m[1], Title: strings.TrimSpace(m[2])})
			cur = len(flat) - 1
			continue
		}
		if m := reNamed.FindStringSubmatch(t); m != nil {
			flat = append(flat, Section{Number: "", Title: m[1]})
			cur = len(flat) - 1
			continue
		}
		if cur >= 0 {
			if flat[cur].Body != "" {
				flat[cur].Body += "\n"
			}
			flat[cur].Body += line
		}
	}
	for i := range flat {
		flat[i].Body = strings.TrimSpace(flat[i].Body)
	}
	if len(flat) == 0 {
		return []Section{{Body: strings.TrimSpace(raw)}}
	}
	return docmodel.Nest(flat)
}
