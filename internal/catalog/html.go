package catalog

import (
	"html"
	"regexp"
	"strings"
)

var (
	reBlock = regexp.MustCompile(`(?i)</p>|<br\s*/?>`)
	reTag   = regexp.MustCompile(`<[^>]*>`)
	reWS    = regexp.MustCompile(`[ \t]+`)
	reNL    = regexp.MustCompile(`\n{3,}`)
)

// StripHTML converts the HTML scope field to readable plain text.
func StripHTML(s string) string {
	s = reBlock.ReplaceAllString(s, "\n\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ") // normalize non-breaking spaces
	s = reWS.ReplaceAllString(s, " ")
	// trim spaces around newlines
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	s = strings.Join(lines, "\n")
	s = reNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
