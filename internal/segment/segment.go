package segment

import (
	"regexp"
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/docmodel"
)

// Section is an alias for the shared document section type.
type Section = docmodel.Section

var reNamed = regexp.MustCompile(`^(Annex\s+[A-Z]|Foreword|Introduction|Scope|Normative references|Bibliography|Terms and definitions)\s*$`)

// Sections splits raw text into nested sections by heading heuristics.
// If no headings are found, returns a single section with the whole text.
func Sections(raw string) []Section {
	return docmodel.BuildSections(raw, func(line string) (string, string, bool) {
		if num, title, ok := docmodel.IsNumberedHeading(line); ok {
			return num, title, true
		}
		if m := reNamed.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			return "", m[1], true
		}
		return "", "", false
	})
}
