// Package docmodel defines shared document and section types used by the
// parse and segment packages.
package docmodel

import "strings"

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
