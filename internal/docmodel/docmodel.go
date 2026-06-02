// Package docmodel defines shared document and section types used by the
// parse and segment packages.
package docmodel

import (
	"regexp"
	"strings"
)

// reNumber matches a heading that begins with an ISO section number, e.g.
// "4.2.1 Title" or "A.1 Title", capturing the number and the remaining title.
var reNumber = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*|[A-Z]\.[0-9]+(?:\.[0-9]+)*)\s+(.+)$`)

// SplitNumber splits a heading line into its leading section number and title.
// If the line has no leading section number, num is "" and title is the
// trimmed input.
func SplitNumber(text string) (num, title string) {
	text = strings.TrimSpace(text)
	if m := reNumber.FindStringSubmatch(text); m != nil {
		return m[1], strings.TrimSpace(m[2])
	}
	return "", text
}

// IsNumberedHeading reports whether line is a standalone numbered heading and,
// if so, returns its number and title.
func IsNumberedHeading(line string) (num, title string, ok bool) {
	if m := reNumber.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
		return m[1], strings.TrimSpace(m[2]), true
	}
	return "", "", false
}

// BuildSections runs the common flat-section accumulation loop over raw text.
// detect is called per line; when it reports a heading, a new section starts
// with the returned number/title, otherwise the line is appended to the
// current section's body. The result is nested with Nest, or a single
// whole-text section when no headings are found.
func BuildSections(raw string, detect func(line string) (num, title string, ok bool)) []Section {
	lines := strings.Split(raw, "\n")
	var flat []Section
	cur := -1
	for _, line := range lines {
		if num, title, ok := detect(line); ok {
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
	return Nest(flat)
}

// Document is a parsed local standards file.
type Document struct {
	Title    string
	Sections []Section
	Raw      string
}

// Section is one chapter/segment, possibly nested.
type Section struct {
	Number   string
	Title    string
	Body     string
	Children []Section
}

// Flatten returns all sections depth-first (used for chapter lookup).
func (d Document) Flatten() []Section {
	var out []Section
	var walk func([]Section)
	walk = func(secs []Section) {
		for _, s := range secs {
			out = append(out, s)
			walk(s.Children)
		}
	}
	walk(d.Sections)
	return out
}

// Nest builds a section tree from a flat list using index-based parent tracking.
// Depth is determined by the number of dots in the section Number field.
// Sections with empty Number are treated as depth 0 (top-level).
func Nest(flat []Section) []Section {
	n := len(flat)
	if n == 0 {
		return nil
	}

	depthOf := func(num string) int {
		if num == "" {
			return 0
		}
		return strings.Count(num, ".")
	}

	// parent[i] = index of parent in flat, -1 = root
	parent := make([]int, n)
	stack := make([]int, 0, n)

	for i, s := range flat {
		d := depthOf(s.Number)
		for len(stack) > 0 && depthOf(flat[stack[len(stack)-1]].Number) >= d {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			parent[i] = -1
		} else {
			parent[i] = stack[len(stack)-1]
		}
		stack = append(stack, i)
	}

	children := make([][]int, n)
	var roots []int
	for i := range flat {
		if parent[i] == -1 {
			roots = append(roots, i)
		} else {
			children[parent[i]] = append(children[parent[i]], i)
		}
	}

	var build func(idx int) Section
	build = func(idx int) Section {
		s := flat[idx]
		s.Children = nil
		for _, ci := range children[idx] {
			s.Children = append(s.Children, build(ci))
		}
		return s
	}

	result := make([]Section, len(roots))
	for i, idx := range roots {
		result[i] = build(idx)
	}
	return result
}
