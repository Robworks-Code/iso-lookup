package parse

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/docmodel"
	"github.com/Robworks-Code/iso-lookup/internal/segment"
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
	return docmodel.BuildSections(raw, func(line string) (string, string, bool) {
		m := reMDHeading.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			return "", "", false
		}
		num, title := docmodel.SplitNumber(m[2])
		return num, title, true
	})
}
