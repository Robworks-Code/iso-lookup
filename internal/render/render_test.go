package render

import (
	"strings"
	"testing"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
)

func sample() catalog.Record {
	return catalog.Record{
		Reference: "ISO/IEC 27001:2022", Title: "ISMS — Requirements",
		Status: "Published", PublishedDate: "2022-10-25", Edition: 3,
		Committee: "ISO/IEC JTC 1/SC 27 — Infosec", ICS: []string{"35.030 IT Security"},
		Scope: "Specifies requirements.", URL: "https://www.iso.org/standard/82875.html",
		Replaces: "ISO/IEC 27001:2013",
	}
}

func TestSummaryContainsKeyFields(t *testing.T) {
	out := Summary(sample())
	for _, want := range []string{"ISO/IEC 27001:2022", "ISMS — Requirements", "Published", "2022-10-25", "ISO/IEC JTC 1/SC 27", "35.030", "Specifies requirements.", "https://www.iso.org/standard/82875.html"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n---\n%s", want, out)
		}
	}
}

func TestSearchListFormat(t *testing.T) {
	out := SearchList([]catalog.Record{sample()})
	if !strings.Contains(out, "ISO/IEC 27001:2022") || !strings.Contains(out, "Published") {
		t.Errorf("search list bad:\n%s", out)
	}
}

func TestActionsNotice(t *testing.T) {
	out := Actions(sample(), false)
	if !strings.Contains(out, "not available locally") || !strings.Contains(out, "iso open") {
		t.Errorf("no-local notice bad:\n%s", out)
	}
	out = Actions(sample(), true)
	if !strings.Contains(out, "Next →") || !strings.Contains(out, "iso browse") {
		t.Errorf("local actions bad:\n%s", out)
	}
}

// A current standard shows no lifecycle caveat; a superseded or withdrawn one does.
func TestSummaryLifecycleCaveat(t *testing.T) {
	if out := Summary(sample()); strings.Contains(out, "⚠") {
		t.Errorf("current standard should not warn:\n%s", out)
	}
	r := sample()
	r.ReplacedBy = "ISO/IEC 27001:2025"
	if out := Summary(r); !strings.Contains(out, "Superseded by ISO/IEC 27001:2025") {
		t.Errorf("superseded standard should warn:\n%s", out)
	}
	r = sample()
	r.Status = "Withdrawn"
	if out := Summary(r); !strings.Contains(out, "Withdrawn") {
		t.Errorf("withdrawn standard should warn:\n%s", out)
	}
}

// shortTitle drops generic boilerplate prefixes and elides very long titles.
func TestShortTitle(t *testing.T) {
	if got := shortTitle("Information technology — Cloud computing — Part 3: Reference architecture"); got != "Cloud computing — Part 3: Reference architecture" {
		t.Errorf("boilerplate prefix not trimmed: %q", got)
	}
	if got := shortTitle("AI management system"); got != "AI management system" {
		t.Errorf("short title should be unchanged: %q", got)
	}
	long := "Systems and software engineering — Systems and software Quality Requirements and Evaluation (SQuaRE) — Product quality model"
	if got := shortTitle(long); !strings.HasSuffix(got, "…") || len([]rune(got)) > shortTitleMax {
		t.Errorf("long title not elided to width: %q (%d runes)", got, len([]rune(got)))
	}
}
