package render

import (
	"fmt"
	"strings"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/parse"
)

func Summary(r catalog.Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", r.Reference, r.Title)
	fmt.Fprintf(&b, "Status:     %s\n", r.Status)
	if r.PublishedDate != "" {
		fmt.Fprintf(&b, "Published:  %s", r.PublishedDate)
		if r.Edition > 0 {
			fmt.Fprintf(&b, " (edition %d)", r.Edition)
		}
		b.WriteString("\n")
	}
	if r.Committee != "" {
		fmt.Fprintf(&b, "Committee:  %s\n", r.Committee)
	}
	if len(r.ICS) > 0 {
		fmt.Fprintf(&b, "ICS:        %s\n", strings.Join(r.ICS, ", "))
	}
	if r.Replaces != "" {
		fmt.Fprintf(&b, "Replaces:   %s\n", r.Replaces)
	}
	if r.ReplacedBy != "" {
		fmt.Fprintf(&b, "Replaced by: %s\n", r.ReplacedBy)
	}
	if r.Pages > 0 {
		fmt.Fprintf(&b, "Pages:      %d\n", r.Pages)
	}
	fmt.Fprintf(&b, "URL:        %s\n", r.URL)
	if r.Scope != "" {
		fmt.Fprintf(&b, "\nScope:\n%s\n", r.Scope)
	}
	return b.String()
}

func SearchList(recs []catalog.Record) string {
	if len(recs) == 0 {
		return "No matches.\n"
	}
	var b strings.Builder
	for _, r := range recs {
		fmt.Fprintf(&b, "%-28s  %-11s  %s\n", r.Reference, r.Status, r.Title)
	}
	return b.String()
}

// SearchListLong is like SearchList but adds publication date and committee
// columns, for when the extra criteria help scan results.
func SearchListLong(recs []catalog.Record) string {
	if len(recs) == 0 {
		return "No matches.\n"
	}
	var b strings.Builder
	for _, r := range recs {
		date := r.PublishedDate
		if date == "" {
			date = "—"
		}
		committee := r.Committee
		if i := strings.Index(committee, " — "); i >= 0 {
			committee = committee[:i] // drop the long descriptive name, keep the code
		}
		fmt.Fprintf(&b, "%-28s  %-11s  %-10s  %-22s  %s\n", r.Reference, r.Status, date, committee, r.Title)
	}
	return b.String()
}

func TOC(doc parse.Document) string {
	var b strings.Builder
	b.WriteString("\nContents:\n")
	var walk func(secs []parse.Section, depth int)
	walk = func(secs []parse.Section, depth int) {
		for _, s := range secs {
			fmt.Fprintf(&b, "%s%s  %s\n", strings.Repeat("  ", depth), s.Number, s.Title)
			walk(s.Children, depth+1)
		}
	}
	walk(doc.Sections, 0)
	return b.String()
}

func NoLocalFile(r catalog.Record) string {
	return fmt.Sprintf("\nFull text not available locally — run `iso open %s` for the official page,\nor add a local copy to your docs folder.\n", r.Reference)
}

func Chapter(s parse.Section) string {
	return fmt.Sprintf("%s  %s\n\n%s\n", s.Number, s.Title, s.Body)
}
