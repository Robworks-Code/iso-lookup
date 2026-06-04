package render

import (
	"fmt"
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/scan"
	"github.com/Robworks-Code/iso-lookup/internal/style"
)

// ScanStack renders just the detected stack: one line per component with its
// evidence files.
func ScanStack(d scan.Detection) string {
	var b strings.Builder
	writeScanHeader(&b, d)
	if len(d.Components) == 0 {
		b.WriteString("\n" + style.Dim.Render("No recognizable components found.") + "\n")
		return b.String()
	}
	b.WriteString("\n" + style.SubHeader.Render("Detected stack:") + "\n")
	writeComponents(&b, d.Components)
	return b.String()
}

// ScanReport renders the full grouped report: the detected stack followed by
// recommended standards grouped per Report.GroupBy. With long, each standard
// gains publication date and committee columns.
func ScanReport(r scan.Report, long bool) string {
	var b strings.Builder
	det := scan.Detection{Root: r.Root, Components: r.Components, Truncated: r.Truncated}
	writeScanHeader(&b, det)

	if len(r.Components) == 0 {
		b.WriteString("\n" + style.Dim.Render("No recognizable components found.") + "\n")
		return b.String()
	}
	b.WriteString("\n" + style.SubHeader.Render("Detected stack:") + "\n")
	writeComponents(&b, r.Components)

	if len(r.Groups) == 0 {
		b.WriteString("\n" + style.Dim.Render("No relevant standards matched. Try --discover for a broader set.") + "\n")
		return b.String()
	}

	stds := 0
	for _, g := range r.Groups {
		b.WriteString("\n")
		writeGroup(&b, g, long)
		stds += len(g.Recommendations)
	}
	summary := fmt.Sprintf("Summary: %d %s → %d %s, %d %s grouped by %s.",
		len(r.Components), plural(len(r.Components), "component"),
		len(r.Groups), plural(len(r.Groups), "group"),
		stds, plural(stds, "standard"), r.GroupBy)
	b.WriteString("\n" + style.Summary.Render(summary) + "\n")
	return b.String()
}

// ScanWhy renders a single group with full rationale and evidence, for the
// `scan why` subcommand.
func ScanWhy(g scan.Group, long bool) string {
	var b strings.Builder
	writeGroup(&b, g, long)
	return b.String()
}

func writeScanHeader(b *strings.Builder, d scan.Detection) {
	line := fmt.Sprintf("Scanned: %s", d.Root)
	if d.FilesSeen > 0 {
		line += style.Dim.Render(fmt.Sprintf("  (%d files, %d %s)", d.FilesSeen, len(d.Components), plural(len(d.Components), "component")))
	}
	b.WriteString(style.Panel.Render(line) + "\n")
	if d.Truncated {
		b.WriteString(style.Warn.Render("note: scan was truncated by a depth or file-count limit; some components may be missing.") + "\n")
	}
}

func writeComponents(b *strings.Builder, comps []scan.Component) {
	for _, c := range comps {
		name := style.Pad(style.Header.Render(c.Name), style.NameW)
		fmt.Fprintf(b, "  %s%s\n", name, style.Dim.Render(strings.Join(c.Evidence, ", ")))
	}
}

func writeGroup(b *strings.Builder, g scan.Group, long bool) {
	header := style.SubHeader.Render(g.Header)
	if g.Total > len(g.Recommendations) {
		header += style.Dim.Render(fmt.Sprintf("  (showing %d of %d)", len(g.Recommendations), g.Total))
	}
	b.WriteString(header + "\n")
	if len(g.Recommendations) == 0 {
		b.WriteString("  " + style.Dim.Render("(no standards in the catalog yet)") + "\n")
	}
	for _, rec := range g.Recommendations {
		writeRecommendation(b, rec, long)
	}
	if len(g.Missing) > 0 {
		fmt.Fprintf(b, "  %s\n", style.Dim.Render("not yet in catalog: "+strings.Join(g.Missing, ", ")))
	}
}

func writeRecommendation(b *strings.Builder, rec scan.Recommendation, long bool) {
	r := rec.Record
	title := r.Title
	if rec.Discovered {
		title += style.Dim.Render("  (discovered)")
	}
	b.WriteString("  ")
	b.WriteString(style.Pad(style.Ref.Render(r.Reference), style.RefW))
	b.WriteString(style.Pad(style.Status(r.Status).Render(r.Status), style.StatusW))
	if long {
		b.WriteString(style.Pad(emDash(r.PublishedDate), style.DateW))
		b.WriteString(style.Pad(style.Dim.Render(committeeCode(r.Committee)), style.CommitteeW))
	}
	b.WriteString(title + "\n")
	if rec.Rationale != "" {
		why := "why: " + rec.Rationale
		if len(rec.Components) > 0 {
			why += "  [" + strings.Join(rec.Components, ", ") + "]"
		}
		b.WriteString("      " + style.Rationale.Render(why) + "\n")
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
