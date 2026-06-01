// Package docmodel defines shared document and section types used by the
// parse and segment packages.
package docmodel

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
