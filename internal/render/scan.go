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
		writeGroup(&b, g, long, r.GroupBy)
		stds += len(g.Recommendations)
	}
	writeStartHere(&b, r)
	summary := fmt.Sprintf("Summary: %d %s → %d %s, %d %s grouped by %s. Advisory starting points, not a compliance checklist.",
		len(r.Components), plural(len(r.Components), "component"),
		len(r.Groups), plural(len(r.Groups), "group"),
		stds, plural(stds, "standard"), r.GroupBy)
	b.WriteString("\n" + style.Summary.Render(summary) + "\n")
	return b.String()
}

// writeStartHere prints a short, prioritized list of next actions: the
// highest-confidence published, curated (non-discovered) standards across all
// groups, deduped by reference, each as a runnable `iso show` command.
func writeStartHere(b *strings.Builder, r scan.Report) {
	const maxStartHere = 5
	var picks []scan.Recommendation
	seen := map[string]bool{}
	for _, want := range []scan.Confidence{scan.High, scan.Medium, scan.Low} {
		for _, g := range r.Groups {
			for _, rec := range g.Recommendations {
				if rec.Confidence != want || rec.Discovered || seen[rec.Record.Reference] {
					continue
				}
				if !strings.Contains(strings.ToLower(rec.Record.Status), "published") {
					continue
				}
				seen[rec.Record.Reference] = true
				picks = append(picks, rec)
			}
		}
	}
	if len(picks) == 0 {
		return
	}
	if len(picks) > maxStartHere {
		picks = picks[:maxStartHere]
	}
	b.WriteString("\n" + style.SubHeader.Render("Start here") + "\n")
	for _, rec := range picks {
		fmt.Fprintf(b, "  %s\n", style.Ref.Render("iso show "+rec.Record.Reference))
	}
}

// ScanWhy renders a single group with full rationale and evidence, for the
// `scan why` subcommand.
func ScanWhy(g scan.Group, long bool) string {
	var b strings.Builder
	writeGroup(&b, g, long, "")
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

// writeGroup renders one group: a header (annotated with the driving components
// unless we're already grouping by component), the distinct rationales stated
// once, then each standard on a confidence-marked line. groupBy is the report's
// grouping ("" for `scan why`, which never annotates the header).
func writeGroup(b *strings.Builder, g scan.Group, long bool, groupBy string) {
	header := style.SubHeader.Render(g.Header)
	if groupBy != scan.GroupByComponent {
		if drivers := groupDrivers(g); drivers != "" {
			header += style.Dim.Render("  from " + drivers)
		}
	}
	if g.Total > len(g.Recommendations) {
		header += style.Dim.Render(fmt.Sprintf("  (showing %d of %d)", len(g.Recommendations), g.Total))
	}
	b.WriteString(header + "\n")
	for _, why := range groupRationales(g) {
		b.WriteString("  " + style.Rationale.Render("why  "+why) + "\n")
	}
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

// groupDrivers is the comma-joined union of the components behind a group's
// recommendations, in first-seen order, for the dim "from …" header annotation.
func groupDrivers(g scan.Group) string {
	var drivers []string
	seen := map[string]bool{}
	for _, rec := range g.Recommendations {
		for _, c := range rec.Components {
			if !seen[c] {
				seen[c] = true
				drivers = append(drivers, c)
			}
		}
	}
	return strings.Join(drivers, ", ")
}

// groupRationales returns the distinct rationales across a group's
// recommendations, in first-seen order, so a shared "why" is stated once
// instead of repeated on every standard line.
func groupRationales(g scan.Group) []string {
	var out []string
	seen := map[string]bool{}
	for _, rec := range g.Recommendations {
		if rec.Rationale == "" || seen[rec.Rationale] {
			continue
		}
		seen[rec.Rationale] = true
		out = append(out, rec.Rationale)
	}
	return out
}

func writeRecommendation(b *strings.Builder, rec scan.Recommendation, long bool) {
	r := rec.Record
	title := r.Title
	if !long {
		title = shortTitle(title)
	}
	if rec.Discovered {
		title += style.Dim.Render("  (discovered)")
	}
	b.WriteString("  " + confidenceMark(rec.Confidence) + " ")
	b.WriteString(style.Pad(style.Ref.Render(r.Reference), style.RefW))
	b.WriteString(style.Pad(style.Status(r.Status).Render(r.Status), style.StatusW))
	if long {
		b.WriteString(style.Pad(emDash(r.PublishedDate), style.DateW))
		b.WriteString(style.Pad(style.Dim.Render(committeeCode(r.Committee)), style.CommitteeW))
	}
	b.WriteString(title + "\n")
}

// confidenceMark returns a filled/half/open dot colored by confidence. The
// glyphs differ by fill, so the signal survives --no-color and Ascii mode.
func confidenceMark(c scan.Confidence) string {
	glyph := "○"
	switch c {
	case scan.High:
		glyph = "●"
	case scan.Medium:
		glyph = "◐"
	}
	return style.Confidence(c.String()).Render(glyph)
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
