package render

import (
	"strings"
	"testing"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
	"github.com/Robworks-Code/iso-lookup/internal/scan"
)

func sampleScanReport() scan.Report {
	rec := catalog.Record{Reference: "ISO/IEC 42001:2023", Title: "AI management system", Status: "Published", PublishedDate: "2023-12-01", Committee: "ISO/IEC JTC 1/SC 42 — AI"}
	return scan.Report{
		Root:    "/proj",
		GroupBy: "component",
		Components: []scan.Component{
			{Name: "OpenAI SDK", Evidence: []string{"requirements.txt"}},
		},
		Groups: []scan.Group{{
			Header: "OpenAI SDK",
			Total:  1,
			Recommendations: []scan.Recommendation{{
				Record:     rec,
				Components: []string{"OpenAI SDK"},
				Rationale:  "ML dependencies bring AI standards into scope.",
				Confidence: scan.High,
			}},
		}},
	}
}

func TestScanReportContainsKeyParts(t *testing.T) {
	out := ScanReport(sampleScanReport(), false)
	for _, want := range []string{"Scanned: /proj", "Detected stack:", "OpenAI SDK", "ISO/IEC 42001:2023", "why", "Start here", "iso show ISO/IEC 42001:2023", "Summary:"} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n---\n%s", want, out)
		}
	}
}

// The rationale is stated once per group, not repeated on every standard line.
func TestScanReportRationaleStatedOnce(t *testing.T) {
	rep := sampleScanReport()
	g := &rep.Groups[0]
	g.Recommendations = append(g.Recommendations, scan.Recommendation{
		Record:     catalog.Record{Reference: "ISO/IEC 22989:2022", Title: "AI concepts and terminology", Status: "Published"},
		Components: []string{"OpenAI SDK"},
		Rationale:  g.Recommendations[0].Rationale,
		Confidence: scan.Medium,
	})
	g.Total = len(g.Recommendations)
	out := ScanReport(rep, false)
	if n := strings.Count(out, "ML dependencies bring AI standards into scope."); n != 1 {
		t.Errorf("shared rationale should appear once, appeared %d times\n---\n%s", n, out)
	}
}

// A confidence marker leads each recommendation line.
func TestScanReportShowsConfidenceMarker(t *testing.T) {
	out := ScanReport(sampleScanReport(), false)
	if !strings.Contains(out, "●") {
		t.Errorf("high-confidence recommendation should carry a filled marker\n---\n%s", out)
	}
}

func TestScanReportLongAddsColumns(t *testing.T) {
	out := ScanReport(sampleScanReport(), true)
	if !strings.Contains(out, "2023-12-01") || !strings.Contains(out, "SC 42") {
		t.Errorf("long report should include date and committee\n%s", out)
	}
}

func TestScanReportNoComponents(t *testing.T) {
	out := ScanReport(scan.Report{Root: "/x"}, false)
	if !strings.Contains(out, "No recognizable components found.") {
		t.Errorf("expected empty-stack message, got:\n%s", out)
	}
}

func TestScanReportNoStandards(t *testing.T) {
	rep := scan.Report{Root: "/x", Components: []scan.Component{{Name: "Go"}}}
	out := ScanReport(rep, false)
	if !strings.Contains(out, "No relevant standards matched") {
		t.Errorf("expected no-standards message, got:\n%s", out)
	}
}

func TestScanStack(t *testing.T) {
	d := scan.Detection{Root: "/proj", FilesSeen: 3, Components: []scan.Component{{Name: "Go", Evidence: []string{"go.mod"}}}}
	out := ScanStack(d)
	if !strings.Contains(out, "Go") || !strings.Contains(out, "go.mod") || strings.Contains(out, "why:") {
		t.Errorf("stack output unexpected:\n%s", out)
	}
}

func TestScanWhyShowsMissing(t *testing.T) {
	g := scan.Group{Header: "Containerization", Missing: []string{"19770"}}
	out := ScanWhy(g, false)
	if !strings.Contains(out, "Containerization") || !strings.Contains(out, "19770") {
		t.Errorf("why output should show header and missing anchors:\n%s", out)
	}
}
