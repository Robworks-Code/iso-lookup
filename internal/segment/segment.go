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
	reNamed    = regexp.MustCompile(`^(Annex\s+[A-Z]|Foreword|Introduction|Scope|Bibliography|Terms and definitions)\s*$`)
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
	return nest(flat)
}

func depthOf(num string) int {
	if num == "" {
		return 0
	}
	// "4" → 0 dots → depth 0 (top-level clause)
	// "4.1" → 1 dot → depth 1
	// "4.1.2" → 2 dots → depth 2
	return strings.Count(num, ".")
}

// nest builds a section tree from a flat list using index-based parent tracking.
// This avoids pointer-into-growing-slice invalidation bugs.
func nest(flat []Section) []Section {
	n := len(flat)
	if n == 0 {
		return nil
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
			p := parent[i]
			children[p] = append(children[p], i)
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
