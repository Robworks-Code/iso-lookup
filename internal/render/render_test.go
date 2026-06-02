package render

import (
	"strings"
	"testing"

	"github.com/ringo380/iso-lookup/internal/catalog"
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

func TestNoLocalFileNotice(t *testing.T) {
	out := NoLocalFile(sample())
	if !strings.Contains(out, "not available locally") || !strings.Contains(out, "iso open") {
		t.Errorf("notice bad:\n%s", out)
	}
}
