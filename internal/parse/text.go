package parse

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ringo380/iso-lookup/internal/docmodel"
	"github.com/ringo380/iso-lookup/internal/segment"
)

// Parse reads a local file and returns a normalized Document, dispatching by
// extension. Unknown extensions are treated as plain text.
func Parse(path string) (Document, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return parseHTML(path)
	case ".pdf":
		return parsePDF(path)
	default:
		return parseText(path)
	}
}

var reMDHeading = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
var reMDNumber = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*|[A-Z]\.[0-9]+(?:\.[0-9]+)*)\s+(.+)$`)

func parseText(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	raw := string(b)
	doc := Document{Raw: raw, Title: filepath.Base(path)}
	if strings.ToLower(filepath.Ext(path)) == ".md" && reMDHeading.MatchString(firstHeading(raw)) {
		doc.Sections = sectionsFromMarkdown(raw)
		return doc, nil
	}
	doc.Sections = segment.Sections(raw)
	return doc, nil
}

func firstHeading(raw string) string {
	for _, l := range strings.Split(raw, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

func sectionsFromMarkdown(raw string) []Section {
	lines := strings.Split(raw, "\n")
	var flat []Section
	cur := -1
	for _, line := range lines {
		if m := reMDHeading.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			text := strings.TrimSpace(m[2])
			num, title := "", text
			if nm := reMDNumber.FindStringSubmatch(text); nm != nil {
				num, title = nm[1], strings.TrimSpace(nm[2])
			}
			flat = append(flat, Section{Number: num, Title: title})
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
