package render

import (
	"fmt"
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/scan"
)

// ScanStack renders just the detected stack: one line per component with its
// evidence files.
func ScanStack(d scan.Detection) string {
	var b strings.Builder
	writeScanHeader(&b, d)
	if len(d.Components) == 0 {
		b.WriteString("\nNo recognizable components found.\n")
		return b.String()
	}
	b.WriteString("\nDetected stack:\n")
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
		b.WriteString("\nNo recognizable components found.\n")
		return b.String()
	}
	b.WriteString("\nDetected stack:\n")
	writeComponents(&b, r.Components)

	if len(r.Groups) == 0 {
		b.WriteString("\nNo relevant standards matched. Try --discover for a broader set.\n")
		return b.String()
	}

	stds := 0
	for _, g := range r.Groups {
		b.WriteString("\n")
		writeGroup(&b, g, long)
		stds += len(g.Recommendations)
	}
	fmt.Fprintf(&b, "\nSummary: %d %s → %d %s, %d %s grouped by %s.\n",
		len(r.Components), plural(len(r.Components), "component"),
		len(r.Groups), plural(len(r.Groups), "group"),
		stds, plural(stds, "standard"), r.GroupBy)
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
	fmt.Fprintf(b, "Scanned: %s", d.Root)
	if d.FilesSeen > 0 {
		fmt.Fprintf(b, "  (%d files, %d %s)", d.FilesSeen, len(d.Components), plural(len(d.Components), "component"))
	}
	b.WriteString("\n")
	if d.Truncated {
		b.WriteString("note: scan was truncated by a depth or file-count limit; some components may be missing.\n")
	}
}

func writeComponents(b *strings.Builder, comps []scan.Component) {
	for _, c := range comps {
		fmt.Fprintf(b, "  %-22s %s\n", c.Name, strings.Join(c.Evidence, ", "))
	}
}

func writeGroup(b *strings.Builder, g scan.Group, long bool) {
	header := g.Header
	if g.Total > len(g.Recommendations) {
		header = fmt.Sprintf("%s  (showing %d of %d)", header, len(g.Recommendations), g.Total)
	}
	b.WriteString(header + "\n")
	if len(g.Recommendations) == 0 {
		b.WriteString("  (no standards in the catalog yet)\n")
	}
	for _, rec := range g.Recommendations {
		writeRecommendation(b, rec, long)
	}
	if len(g.Missing) > 0 {
		fmt.Fprintf(b, "  not yet in catalog: %s\n", strings.Join(g.Missing, ", "))
	}
}

func writeRecommendation(b *strings.Builder, rec scan.Recommendation, long bool) {
	r := rec.Record
	tag := ""
	if rec.Discovered {
		tag = "  (discovered)"
	}
	if long {
		date := r.PublishedDate
		if date == "" {
			date = "—"
		}
		committee := r.Committee
		if i := strings.Index(committee, " — "); i >= 0 {
			committee = committee[:i]
		}
		fmt.Fprintf(b, "  %-28s  %-11s  %-10s  %-22s  %s%s\n", r.Reference, r.Status, date, committee, r.Title, tag)
	} else {
		fmt.Fprintf(b, "  %-28s  %-11s  %s%s\n", r.Reference, r.Status, r.Title, tag)
	}
	if rec.Rationale != "" {
		driven := ""
		if len(rec.Components) > 0 {
			driven = "  [" + strings.Join(rec.Components, ", ") + "]"
		}
		fmt.Fprintf(b, "      why: %s%s\n", rec.Rationale, driven)
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
