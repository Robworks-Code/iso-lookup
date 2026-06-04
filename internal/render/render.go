package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
	"github.com/Robworks-Code/iso-lookup/internal/config"
	"github.com/Robworks-Code/iso-lookup/internal/parse"
	"github.com/Robworks-Code/iso-lookup/internal/style"
)

// scopeWidth bounds wrapped prose (scope text) so long paragraphs stay readable.
const scopeWidth = 96

func Summary(r catalog.Record) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", style.Ref.Render(r.Reference), style.Header.Render(r.Title))
	writeField(&b, "Status:", style.Status(r.Status).Render(r.Status))
	if r.PublishedDate != "" {
		date := r.PublishedDate
		if r.Edition > 0 {
			date = fmt.Sprintf("%s (edition %d)", date, r.Edition)
		}
		writeField(&b, "Published:", date)
	}
	if r.Committee != "" {
		writeField(&b, "Committee:", r.Committee)
	}
	if len(r.ICS) > 0 {
		writeField(&b, "ICS:", strings.Join(r.ICS, ", "))
	}
	if r.Replaces != "" {
		writeField(&b, "Replaces:", r.Replaces)
	}
	if r.ReplacedBy != "" {
		writeField(&b, "Replaced by:", r.ReplacedBy)
	}
	if r.Pages > 0 {
		writeField(&b, "Pages:", fmt.Sprintf("%d", r.Pages))
	}
	writeField(&b, "URL:", style.URL.Render(r.URL))
	if r.Scope != "" {
		fmt.Fprintf(&b, "\n%s\n%s\n", style.SubHeader.Render("Scope"),
			lipgloss.NewStyle().Width(scopeWidth).Render(r.Scope))
	}
	return b.String()
}

// writeField prints a grey, fixed-width label followed by its value so the
// values line up regardless of label length.
func writeField(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "%s %s\n", style.Label.Render(fmt.Sprintf("%-12s", label)), value)
}

func SearchList(recs []catalog.Record) string {
	if len(recs) == 0 {
		return style.Dim.Render("No matches.") + "\n"
	}
	var b strings.Builder
	for _, r := range recs {
		b.WriteString(style.Pad(style.Ref.Render(r.Reference), style.RefW))
		b.WriteString(style.Pad(style.Status(r.Status).Render(r.Status), style.StatusW))
		b.WriteString(r.Title + "\n")
	}
	return b.String()
}

// SearchListLong is like SearchList but adds publication date and committee
// columns, for when the extra criteria help scan results.
func SearchListLong(recs []catalog.Record) string {
	if len(recs) == 0 {
		return style.Dim.Render("No matches.") + "\n"
	}
	var b strings.Builder
	for _, r := range recs {
		b.WriteString(style.Pad(style.Ref.Render(r.Reference), style.RefW))
		b.WriteString(style.Pad(style.Status(r.Status).Render(r.Status), style.StatusW))
		b.WriteString(style.Pad(emDash(r.PublishedDate), style.DateW))
		b.WriteString(style.Pad(style.Dim.Render(committeeCode(r.Committee)), style.CommitteeW))
		b.WriteString(r.Title + "\n")
	}
	return b.String()
}

// committeeCode keeps only the committee code, dropping the long descriptive
// name after the em-dash separator.
func committeeCode(committee string) string {
	if i := strings.Index(committee, " — "); i >= 0 {
		return committee[:i]
	}
	return committee
}

// emDash substitutes a dim em-dash for an empty value.
func emDash(s string) string {
	if s == "" {
		return style.Dim.Render("—")
	}
	return s
}

func TOC(doc parse.Document) string {
	var b strings.Builder
	b.WriteString("\n" + style.SubHeader.Render("Contents") + "\n")
	var walk func(secs []parse.Section, depth int)
	walk = func(secs []parse.Section, depth int) {
		for _, s := range secs {
			fmt.Fprintf(&b, "%s%s  %s\n", strings.Repeat("  ", depth), style.Dim.Render(s.Number), s.Title)
			walk(s.Children, depth+1)
		}
	}
	walk(doc.Sections, 0)
	return b.String()
}

func NoLocalFile(r catalog.Record) string {
	return "\n" + style.Dim.Render("Full text not available locally — run ") +
		style.Ref.Render("iso open "+r.Reference) +
		style.Dim.Render(" for the official page,\nor add a local copy to your docs folder.") + "\n"
}

func Chapter(s parse.Section) string {
	return fmt.Sprintf("%s\n\n%s\n", style.SubHeader.Render(s.Number+"  "+s.Title), s.Body)
}

// Config renders the current configuration as aligned grey-labelled fields,
// showing a dim "(not set)" for unset values.
func Config(c config.Config) string {
	var b strings.Builder
	writeField(&b, "docs_dir:", orNotSet(c.DocsDir))
	writeField(&b, "index_file:", orNotSet(c.IndexFile))
	writeField(&b, "pager:", orNotSet(c.Pager))
	return b.String()
}

func orNotSet(s string) string {
	if s == "" {
		return style.Dim.Render("(not set)")
	}
	return s
}
