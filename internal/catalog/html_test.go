package catalog

import "testing"

func TestStripHTML(t *testing.T) {
	in := "<p>This document specifies requirements.</p>\n<p>a) within an ISMS based on ISO/IEC&nbsp;27001;</p><br/>"
	want := "This document specifies requirements.\n\na) within an ISMS based on ISO/IEC 27001;"
	if got := StripHTML(in); got != want {
		t.Fatalf("StripHTML mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestStripHTMLPlainPassesThrough(t *testing.T) {
	if got := StripHTML("plain text"); got != "plain text" {
		t.Fatalf("got %q", got)
	}
}
