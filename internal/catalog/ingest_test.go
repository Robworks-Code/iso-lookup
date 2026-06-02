package catalog

import (
	"os"
	"testing"
)

func TestIngest(t *testing.T) {
	del, _ := os.Open("testdata/deliverables.jsonl")
	defer del.Close()
	com, _ := os.Open("testdata/committees.jsonl")
	defer com.Close()
	ics, _ := os.Open("testdata/ICS.csv")
	defer ics.Close()

	recs, err := Ingest(del, com, ics)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	var r Record
	for _, x := range recs {
		if x.Reference == "ISO/IEC 27001:2022" {
			r = x
		}
	}
	if r.Title == "" {
		t.Fatal("27001 not found")
	}
	if r.Scope != "This document specifies the requirements." {
		t.Errorf("scope = %q", r.Scope)
	}
	if r.Status != "Published" {
		t.Errorf("status = %q", r.Status)
	}
	if r.URL != "https://www.iso.org/standard/82875.html" {
		t.Errorf("url = %q", r.URL)
	}
	if r.Committee != "ISO/IEC JTC 1/SC 27 — Information security, cybersecurity and privacy protection" {
		t.Errorf("committee = %q", r.Committee)
	}
	if len(r.ICS) != 2 || r.ICS[0] != "35.030 IT Security" {
		t.Errorf("ics = %v", r.ICS)
	}
	if r.Pages != 19 {
		t.Errorf("pages = %d", r.Pages)
	}
}
