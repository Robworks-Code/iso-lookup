# iso-lookup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Go CLI (`iso`) that looks up security/standards docs by keyword or ISO reference, prints authoritative metadata up front from the offline ISO Open Data index, and reads/navigates full text from local files when available.

**Architecture:** Layered packages behind interfaces. `catalog` ingests the official ISO Open Data JSONL into a slim offline gob index (search + lookup). `library` maps a reference to a local file. `parse`+`segment` turn local PDF/text/MD/HTML into a normalized `Document` with sections. `render` prints to stdout; `tui` browses interactively. Metadata works for every standard offline; full text comes only from user-provided files.

**Tech Stack:** Go 1.25, cobra (CLI), bubbletea+lipgloss (TUI), ledongthuc/pdf (PDF text), golang.org/x/net/html (HTML), stdlib (JSONL, gob, in-memory search).

**Module path:** `github.com/ringo380/iso-lookup` · **Binary:** `iso`

**Reference facts (verified 2026-06-01):**
- Open Data base: `https://isopublicstorageprod.blob.core.windows.net/opendata/_latest`
  - `iso_deliverables_metadata/json/iso_deliverables_metadata.jsonl` (~80,726 records)
  - `iso_technical_committees/json/iso_technical_committees.jsonl`
  - `iso_ics/csv/ICS.csv`
- Deliverable fields: `id`, `reference`, `title.en`, `scope.en` (HTML), `edition`, `publicationDate`, `icsCode[]`, `ownerCommittee`, `currentStage`, `replaces`, `replacedBy`, `pages.en`.
- Official URL: `https://www.iso.org/standard/{id}.html` (the numeric `id`, NOT the reference number).
- `currentStage` = stage*100+substage (6060 = "Published", 9599 = "Withdrawn").
- Committee dataset keyed by `reference` (e.g. `ISO/IEC JTC 1/SC 27`) → `title.en`.
- ICS CSV columns: `identifier,parent,titleEn,titleFr,scopeEn,scopeFr`.
- License: ODC-By 1.0 — must attribute ISO.

---

## File Structure

```
go.mod
cmd/iso/main.go              # cobra root, wires commands
internal/config/config.go    # XDG paths, config.yaml load/save
internal/catalog/record.go   # Record type
internal/catalog/stage.go    # stage-code -> label
internal/catalog/html.go     # strip HTML scope -> plain text
internal/catalog/ingest.go   # JSONL -> []Record (+ committee/ICS resolution)
internal/catalog/store.go    # gob save/load index
internal/catalog/catalog.go  # Catalog: Lookup + Search over loaded index
internal/catalog/download.go # fetch Open Data files (injectable client)
internal/library/library.go  # convention + index.yaml -> file path
internal/parse/document.go   # Document/Section types + Parse dispatch
internal/parse/text.go       # txt/md parser
internal/parse/html.go       # html parser
internal/parse/pdf.go        # pdf parser
internal/segment/segment.go  # heading/numbering heuristics -> []Section
internal/render/render.go    # summary, search list, chapter, json
internal/tui/tui.go          # bubbletea browser
cmd/iso/cmd_*.go             # one file per command
```

Tests live beside each file as `*_test.go`. Fixtures under `internal/<pkg>/testdata/`.

---

## Task 0: Project scaffold

**Files:**
- Create: `go.mod`, `cmd/iso/main.go`

- [ ] **Step 1: Initialize the module**

```bash
cd /Users/ryanrobson/git/iso-lookup
go mod init github.com/ringo380/iso-lookup
go get github.com/spf13/cobra@latest
```

- [ ] **Step 2: Write the cobra root in `cmd/iso/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "iso",
	Short: "Look up ISO security and standards documents",
	Long:  "iso looks up standards by keyword or reference, prints metadata from the offline ISO Open Data index, and reads full text from local files when available.",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify it builds and runs**

Run: `go build -o iso ./cmd/iso && ./iso --help`
Expected: prints usage with "iso" and the long description.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum cmd/iso/main.go
git commit -m "feat: scaffold iso CLI with cobra root"
```

---

## Task 1: config package (paths)

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirRespectsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdgtest")
	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/xdgtest", "iso-lookup")
	if got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), "iso-cfg-missing"))
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DocsDir != "" {
		t.Fatalf("expected empty DocsDir default, got %q", c.DocsDir)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(os.TempDir(), "iso-cfg-rt"))
	in := Config{DocsDir: "/docs", IndexFile: "/docs/index.yaml", Pager: "less"}
	if err := Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch: %+v vs %+v", out, in)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/`
Expected: FAIL (package/functions undefined).

- [ ] **Step 3: Implement `config.go`**

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user settings persisted to config.json.
type Config struct {
	DocsDir   string `json:"docs_dir"`
	IndexFile string `json:"index_file"`
	Pager     string `json:"pager"`
}

// Dir returns the iso-lookup config/cache directory (XDG-aware).
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "iso-lookup"), nil
}

func configPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// CachePath returns the path to the built gob index.
func CachePath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "catalog.gob"), nil
}

// Load reads config.json, returning a zero-value Config if it does not exist.
func Load() (Config, error) {
	p, err := configPath()
	if err != nil {
		return Config{}, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes config.json, creating the directory if needed.
func Save(c Config) error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	p := filepath.Join(d, "config.json")
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): XDG-aware config load/save"
```

---

## Task 2: catalog Record + stage-code labels

**Files:**
- Create: `internal/catalog/record.go`, `internal/catalog/stage.go`, `internal/catalog/stage_test.go`

- [ ] **Step 1: Write the failing test for stage labels**

```go
package catalog

import "testing"

func TestStageLabel(t *testing.T) {
	cases := map[int]string{
		6060: "Published",
		9599: "Withdrawn",
		4020: "Enquiry",          // stage-group fallback (40.20)
		1234: "Stage 12.34",      // unknown -> raw
	}
	for code, want := range cases {
		if got := StageLabel(code); got != want {
			t.Errorf("StageLabel(%d) = %q, want %q", code, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestStageLabel`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `record.go`**

```go
package catalog

// Record is the slim, display-ready metadata for one ISO deliverable.
type Record struct {
	Reference     string
	Title         string
	Scope         string // HTML stripped to plain text
	Edition       int
	PublishedDate string
	StageCode     int
	Status        string // StageLabel(StageCode)
	ICS           []string
	Committee     string
	Replaces      string
	ReplacedBy    string
	Pages         int
	ID            int
	URL           string
}
```

- [ ] **Step 4: Implement `stage.go`**

```go
package catalog

import "fmt"

// exactStage maps full ISO harmonized stage codes (stage*100+substage) to labels.
var exactStage = map[int]string{
	6060: "Published",
	6000: "Publication",
	9599: "Withdrawn",
	9060: "Review completed",
	9020: "Under review",
	1099: "New project approved",
}

// stageGroup maps the stage prefix (code/100) to a coarse label.
var stageGroup = map[int]string{
	0:  "Preliminary",
	10: "Proposal",
	20: "Preparatory",
	30: "Committee",
	40: "Enquiry",
	50: "Approval",
	60: "Publication",
	90: "Review",
	95: "Withdrawal",
}

// StageLabel converts an ISO stage code to a human-readable status.
func StageLabel(code int) string {
	if s, ok := exactStage[code]; ok {
		return s
	}
	if s, ok := stageGroup[code/100]; ok {
		return s
	}
	return fmt.Sprintf("Stage %02d.%02d", code/100, code%100)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestStageLabel`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/catalog/record.go internal/catalog/stage.go internal/catalog/stage_test.go
git commit -m "feat(catalog): Record type and stage-code labels"
```

---

## Task 3: HTML scope stripping

**Files:**
- Create: `internal/catalog/html.go`, `internal/catalog/html_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestStripHTML`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `html.go`** (stdlib only; `<p>`/`<br>` become breaks, other tags drop, entities unescaped, whitespace trimmed)

```go
package catalog

import (
	"html"
	"regexp"
	"strings"
)

var (
	reBlock = regexp.MustCompile(`(?i)</p>|<br\s*/?>`)
	reTag   = regexp.MustCompile(`<[^>]*>`)
	reWS    = regexp.MustCompile(`[ \t]+`)
	reNL    = regexp.MustCompile(`\n{3,}`)
)

// StripHTML converts the HTML scope field to readable plain text.
func StripHTML(s string) string {
	s = reBlock.ReplaceAllString(s, "\n\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ") // nbsp
	s = reWS.ReplaceAllString(s, " ")
	// trim spaces around newlines
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	s = strings.Join(lines, "\n")
	s = reNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestStripHTML`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/html.go internal/catalog/html_test.go
git commit -m "feat(catalog): strip HTML from scope text"
```

---

## Task 4: ingest JSONL (with committee + ICS resolution)

**Files:**
- Create: `internal/catalog/ingest.go`, `internal/catalog/ingest_test.go`, `internal/catalog/testdata/deliverables.jsonl`, `internal/catalog/testdata/committees.jsonl`, `internal/catalog/testdata/ICS.csv`

- [ ] **Step 1: Create fixtures**

`internal/catalog/testdata/deliverables.jsonl` (two lines):

```json
{"id":82875,"reference":"ISO/IEC 27001:2022","title":{"en":"Information security management systems — Requirements"},"scope":{"en":"<p>This document specifies the requirements.</p>"},"edition":3,"publicationDate":"2022-10-25","icsCode":["35.030","03.100.70"],"ownerCommittee":"ISO/IEC JTC 1/SC 27","currentStage":6060,"replaces":"ISO/IEC 27001:2013","replacedBy":null,"pages":{"en":19}}
{"id":1,"reference":"ISO/WD 0","title":{"en":"Road vehicles"},"scope":{"en":null},"edition":1,"publicationDate":null,"icsCode":null,"ownerCommittee":"ISO/TC 22","currentStage":2020,"replaces":null,"replacedBy":null,"pages":{"en":null}}
```

`internal/catalog/testdata/committees.jsonl`:

```json
{"reference":"ISO/IEC JTC 1/SC 27","title":{"en":"Information security, cybersecurity and privacy protection"}}
{"reference":"ISO/TC 22","title":{"en":"Road vehicles"}}
```

`internal/catalog/testdata/ICS.csv`:

```csv
identifier,parent,titleEn,titleFr,scopeEn,scopeFr
"35.030",,"IT Security","Sécurité IT",,
"03.100.70",,"Management systems","Systèmes de management",,
```

- [ ] **Step 2: Write the failing test**

```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestIngest`
Expected: FAIL (undefined `Ingest`).

- [ ] **Step 4: Implement `ingest.go`**

```go
package catalog

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// rawDeliverable mirrors the Open Data deliverables JSONL shape.
type rawDeliverable struct {
	ID             int               `json:"id"`
	Reference      string            `json:"reference"`
	Title          map[string]string `json:"title"`
	Scope          map[string]string `json:"scope"`
	Edition        int               `json:"edition"`
	PublicationDate string           `json:"publicationDate"`
	ICSCode        []string          `json:"icsCode"`
	OwnerCommittee string            `json:"ownerCommittee"`
	CurrentStage   int               `json:"currentStage"`
	Replaces       string            `json:"replaces"`
	ReplacedBy     string            `json:"replacedBy"`
	Pages          map[string]*int   `json:"pages"`
}

type rawCommittee struct {
	Reference string            `json:"reference"`
	Title     map[string]string `json:"title"`
}

// Ingest parses the three Open Data sources into slim Records.
func Ingest(deliverables, committees, ics io.Reader) ([]Record, error) {
	comNames, err := parseCommittees(committees)
	if err != nil {
		return nil, fmt.Errorf("committees: %w", err)
	}
	icsNames, err := parseICS(ics)
	if err != nil {
		return nil, fmt.Errorf("ics: %w", err)
	}

	var recs []Record
	sc := bufio.NewScanner(deliverables)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var d rawDeliverable
		if err := json.Unmarshal(line, &d); err != nil {
			return nil, err
		}
		recs = append(recs, toRecord(d, comNames, icsNames))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return recs, nil
}

func toRecord(d rawDeliverable, com, ics map[string]string) Record {
	r := Record{
		Reference:     d.Reference,
		Title:         d.Title["en"],
		Scope:         StripHTML(d.Scope["en"]),
		Edition:       d.Edition,
		PublishedDate: d.PublicationDate,
		StageCode:     d.CurrentStage,
		Status:        StageLabel(d.CurrentStage),
		Replaces:      d.Replaces,
		ReplacedBy:    d.ReplacedBy,
		ID:            d.ID,
		URL:           fmt.Sprintf("https://www.iso.org/standard/%d.html", d.ID),
	}
	if p := d.Pages["en"]; p != nil {
		r.Pages = *p
	}
	if name := com[d.OwnerCommittee]; name != "" {
		r.Committee = d.OwnerCommittee + " — " + name
	} else {
		r.Committee = d.OwnerCommittee
	}
	for _, code := range d.ICSCode {
		if name := ics[code]; name != "" {
			r.ICS = append(r.ICS, code+" "+name)
		} else {
			r.ICS = append(r.ICS, code)
		}
	}
	return r
}

func parseCommittees(rd io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var c rawCommittee
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			return nil, err
		}
		out[c.Reference] = c.Title["en"]
	}
	return out, sc.Err()
}

func parseICS(rd io.Reader) (map[string]string, error) {
	out := map[string]string{}
	r := csv.NewReader(rd)
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		if i == 0 || len(row) < 3 { // skip header
			continue
		}
		id := strings.TrimPrefix(row[0], "﻿") // strip BOM on first data cell if present
		out[id] = row[2]
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestIngest`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/catalog/ingest.go internal/catalog/ingest_test.go internal/catalog/testdata/
git commit -m "feat(catalog): ingest Open Data JSONL with committee/ICS resolution"
```

---

## Task 5: gob index store

**Files:**
- Create: `internal/catalog/store.go`, `internal/catalog/store_test.go`

- [ ] **Step 1: Write the failing test**

```go
package catalog

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadIndex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "catalog.gob")
	in := []Record{{Reference: "ISO/IEC 27001:2022", Title: "ISMS"}}
	if err := SaveIndex(p, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Reference != in[0].Reference {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestLoadMissingIndex(t *testing.T) {
	_, err := LoadIndex(filepath.Join(t.TempDir(), "nope.gob"))
	if err != ErrNoIndex {
		t.Fatalf("want ErrNoIndex, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestSaveLoadIndex`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `store.go`**

```go
package catalog

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
)

// ErrNoIndex is returned by LoadIndex when no built index exists yet.
var ErrNoIndex = errors.New("no catalog index found; run `iso update`")

// SaveIndex writes records to path as gob, creating parent dirs.
func SaveIndex(path string, recs []Record) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(recs)
}

// LoadIndex reads the gob index, returning ErrNoIndex if absent.
func LoadIndex(path string) ([]Record, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, ErrNoIndex
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var recs []Record
	if err := gob.NewDecoder(f).Decode(&recs); err != nil {
		return nil, err
	}
	return recs, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run 'TestSaveLoadIndex|TestLoadMissingIndex'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/store.go internal/catalog/store_test.go
git commit -m "feat(catalog): gob index save/load"
```

---

## Task 6: Catalog Lookup + Search

**Files:**
- Create: `internal/catalog/catalog.go`, `internal/catalog/catalog_test.go`

- [ ] **Step 1: Write the failing test**

```go
package catalog

import "testing"

func newTestCatalog() *Catalog {
	return New([]Record{
		{Reference: "ISO/IEC 27001:2022", Title: "Information security management systems — Requirements", Scope: "specifies requirements for an ISMS", ReplacedBy: ""},
		{Reference: "ISO/IEC 27001:2013", Title: "Information security management systems — Requirements", ReplacedBy: "ISO/IEC 27001:2022"},
		{Reference: "ISO 9001:2015", Title: "Quality management systems — Requirements", Scope: "quality management"},
	})
}

func TestLookupExact(t *testing.T) {
	c := newTestCatalog()
	r, ok := c.Lookup("ISO/IEC 27001:2022")
	if !ok || r.Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("exact lookup failed: %+v ok=%v", r, ok)
	}
}

func TestLookupBareNumberPrefersCurrent(t *testing.T) {
	c := newTestCatalog()
	r, ok := c.Lookup("27001")
	if !ok {
		t.Fatal("bare-number lookup failed")
	}
	if r.Reference != "ISO/IEC 27001:2022" {
		t.Fatalf("expected current (non-replaced) edition, got %q", r.Reference)
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	c := newTestCatalog()
	if _, ok := c.Lookup("iso 9001"); !ok {
		t.Fatal("case-insensitive lookup failed")
	}
}

func TestSearchRanksReferenceThenTitleThenScope(t *testing.T) {
	c := newTestCatalog()
	res := c.Search("management")
	if len(res) == 0 {
		t.Fatal("no results")
	}
	// 9001 has "management" in title; 27001:2022 has it in title too.
	// All matches returned; ensure quality matches present.
	found := false
	for _, r := range res {
		if r.Reference == "ISO 9001:2015" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ISO 9001:2015 in results")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run 'TestLookup|TestSearch'`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `catalog.go`**

```go
package catalog

import (
	"regexp"
	"sort"
	"strings"
)

// Catalog provides offline lookup/search over loaded Records.
type Catalog struct {
	records []Record
	byRef   map[string]int // normalized reference -> index
}

// New builds a Catalog and its lookup index.
func New(recs []Record) *Catalog {
	c := &Catalog{records: recs, byRef: make(map[string]int, len(recs))}
	for i, r := range recs {
		c.byRef[normalize(r.Reference)] = i
	}
	return c
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return s
}

var reBareNum = regexp.MustCompile(`(\d{3,5})`)

// Lookup resolves a reference exactly, then loosely. For a bare number it
// prefers the non-replaced (current) edition.
func (c *Catalog) Lookup(ref string) (Record, bool) {
	if i, ok := c.byRef[normalize(ref)]; ok {
		return c.records[i], true
	}
	// "iso 9001" with no edition, or bare "27001": match by number.
	m := reBareNum.FindString(ref)
	if m == "" {
		return Record{}, false
	}
	var matches []Record
	for _, r := range c.records {
		if strings.Contains(r.Reference, m) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return Record{}, false
	}
	// prefer current (ReplacedBy == ""), then latest by reference string desc.
	sort.SliceStable(matches, func(a, b int) bool {
		ca, cb := matches[a].ReplacedBy == "", matches[b].ReplacedBy == ""
		if ca != cb {
			return ca // current first
		}
		return matches[a].Reference > matches[b].Reference
	})
	return matches[0], true
}

// Search returns records matching all query tokens, ranked
// reference > title > scope.
func (c *Catalog) Search(query string) []Record {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return nil
	}
	type scored struct {
		r     Record
		score int
	}
	var hits []scored
	for _, r := range c.records {
		ref := strings.ToLower(r.Reference)
		title := strings.ToLower(r.Title)
		scope := strings.ToLower(r.Scope)
		ok := true
		score := 0
		for _, tok := range tokens {
			switch {
			case strings.Contains(ref, tok):
				score += 3
			case strings.Contains(title, tok):
				score += 2
			case strings.Contains(scope, tok):
				score += 1
			default:
				ok = false
			}
		}
		if ok {
			hits = append(hits, scored{r, score})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score > hits[b].score
		}
		return hits[a].r.Reference < hits[b].r.Reference
	})
	out := make([]Record, len(hits))
	for i, h := range hits {
		out[i] = h.r
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/catalog/ -run 'TestLookup|TestSearch'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/catalog.go internal/catalog/catalog_test.go
git commit -m "feat(catalog): offline Lookup and ranked Search"
```

---

## Task 7: download + `iso update` command

**Files:**
- Create: `internal/catalog/download.go`, `internal/catalog/download_test.go`, `cmd/iso/cmd_update.go`

- [ ] **Step 1: Write the failing test (injectable client)**

```go
package catalog

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	}))
	defer srv.Close()
	body, err := fetchURL(srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	b, _ := io.ReadAll(body)
	if strings.TrimSpace(string(b)) != "hello" {
		t.Fatalf("got %q", b)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestFetchURL`
Expected: FAIL (undefined `fetchURL`).

- [ ] **Step 3: Implement `download.go`**

```go
package catalog

import (
	"fmt"
	"io"
	"net/http"
)

const openDataBase = "https://isopublicstorageprod.blob.core.windows.net/opendata/_latest"

// URLs for the three Open Data sources.
var (
	DeliverablesURL = openDataBase + "/iso_deliverables_metadata/json/iso_deliverables_metadata.jsonl"
	CommitteesURL   = openDataBase + "/iso_technical_committees/json/iso_technical_committees.jsonl"
	ICSURL          = openDataBase + "/iso_ics/csv/ICS.csv"
)

func fetchURL(client *http.Client, url string) (io.ReadCloser, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

// BuildIndex downloads all three datasets and returns ingested Records.
func BuildIndex(client *http.Client) ([]Record, error) {
	del, err := fetchURL(client, DeliverablesURL)
	if err != nil {
		return nil, err
	}
	defer del.Close()
	com, err := fetchURL(client, CommitteesURL)
	if err != nil {
		return nil, err
	}
	defer com.Close()
	ics, err := fetchURL(client, ICSURL)
	if err != nil {
		return nil, err
	}
	defer ics.Close()
	return Ingest(del, com, ics)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestFetchURL`
Expected: PASS.

- [ ] **Step 5: Implement `cmd/iso/cmd_update.go`**

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download/refresh the ISO Open Data metadata index",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Downloading ISO Open Data (this may take a moment)...")
		client := &http.Client{Timeout: 5 * time.Minute}
		recs, err := catalog.BuildIndex(client)
		if err != nil {
			return fmt.Errorf("build index (existing index left untouched): %w", err)
		}
		path, err := config.CachePath()
		if err != nil {
			return err
		}
		if err := catalog.SaveIndex(path, recs); err != nil {
			return err
		}
		fmt.Printf("Indexed %d ISO deliverables -> %s\n", len(recs), path)
		fmt.Println("Data © ISO, via the ISO Open Data initiative, licensed under ODC-By 1.0.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
```

- [ ] **Step 6: Build, then run a real update (integration smoke)**

Run: `go build -o iso ./cmd/iso && ./iso update`
Expected: prints "Indexed 80xxx ISO deliverables" and the attribution line. (Requires network.)

- [ ] **Step 7: Commit**

```bash
git add internal/catalog/download.go internal/catalog/download_test.go cmd/iso/cmd_update.go
git commit -m "feat(catalog): Open Data download and iso update command"
```

---

## Task 8: render package

**Files:**
- Create: `internal/render/render.go`, `internal/render/render_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `render.go`**

```go
package render

import (
	"fmt"
	"strings"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/parse"
)

// Summary renders the metadata block printed "up front".
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
		fmt.Fprintf(&b, "Replaced by:%s\n", r.ReplacedBy)
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

// SearchList renders a compact match list.
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

// TOC renders a numbered table of contents from a parsed document.
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

// NoLocalFile renders the notice shown when no local copy is mapped.
func NoLocalFile(r catalog.Record) string {
	return fmt.Sprintf("\nFull text not available locally — run `iso open %s` for the official page,\nor add a local copy to your docs folder.\n", refForOpen(r))
}

func refForOpen(r catalog.Record) string {
	return r.Reference
}

// Chapter renders a single section body.
func Chapter(s parse.Section) string {
	return fmt.Sprintf("%s  %s\n\n%s\n", s.Number, s.Title, s.Body)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/render/`
Expected: PASS. (Note: `parse.Document`/`Section` are defined in Task 9; if running this task first, the test compiles against them — implement Task 9's `document.go` types before this build. Reorder: do Task 9 Step "types" first, then return. See Task 9.)

- [ ] **Step 5: Commit**

```bash
git add internal/render/
git commit -m "feat(render): summary, search list, TOC, chapter, notice"
```

---

## Task 9: parse types + text/markdown parser + segment

> Render (Task 8) imports `parse` types. Implement `internal/parse/document.go` (types only) and `internal/segment/segment.go` before building Task 8.

**Files:**
- Create: `internal/parse/document.go`, `internal/segment/segment.go`, `internal/segment/segment_test.go`, `internal/parse/text.go`, `internal/parse/text_test.go`, `internal/parse/testdata/sample.md`, `internal/parse/testdata/sample.txt`

- [ ] **Step 1: Implement `internal/parse/document.go` (types only)**

```go
package parse

// Document is a parsed local standards file.
type Document struct {
	Title    string
	Sections []Section
	Raw      string
}

// Section is one chapter/segment, possibly nested.
type Section struct {
	Number   string
	Title    string
	Body     string
	Children []Section
}

// Flatten returns all sections depth-first (used for chapter lookup).
func (d Document) Flatten() []Section {
	var out []Section
	var walk func([]Section)
	walk = func(secs []Section) {
		for _, s := range secs {
			out = append(out, s)
			walk(s.Children)
		}
	}
	walk(d.Sections)
	return out
}
```

- [ ] **Step 2: Write failing test for segment**

`internal/segment/segment_test.go`:

```go
package segment

import "testing"

func TestSectionsNumberedHeadings(t *testing.T) {
	raw := "Foreword\n\nIntro text.\n\n1 Scope\n\nThis clause.\n\n4 Context\n\n4.1 Understanding\n\nDetails.\n\nAnnex A\n\nA.5 Controls\n\nMore."
	secs := Sections(raw)
	if len(secs) == 0 {
		t.Fatal("no sections")
	}
	// find "4" with child "4.1"
	var four *Section
	for i := range secs {
		if secs[i].Number == "4" {
			four = &secs[i]
		}
	}
	if four == nil {
		t.Fatal("clause 4 not found")
	}
	if len(four.Children) != 1 || four.Children[0].Number != "4.1" {
		t.Fatalf("expected 4.1 child, got %+v", four.Children)
	}
}

func TestSectionsNoStructureFallback(t *testing.T) {
	secs := Sections("just a blob of text with no headings at all")
	if len(secs) != 1 || secs[0].Number != "" {
		t.Fatalf("expected single fallback section, got %+v", secs)
	}
}
```

(Note: `segment.Section` is an alias of `parse.Section` — see Step 4.)

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/segment/`
Expected: FAIL (undefined).

- [ ] **Step 4: Implement `internal/segment/segment.go`**

```go
package segment

import (
	"regexp"
	"strings"

	"github.com/ringo380/iso-lookup/internal/parse"
)

// Section is re-exported from parse for convenience.
type Section = parse.Section

// reHeading matches numbered clauses (1, 4.1, A.5) and named annex markers.
var (
	reNumbered = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*|[A-Z]\.[0-9]+(?:\.[0-9]+)*)\s+(.+)$`)
	reNamed    = regexp.MustCompile(`^(Annex\s+[A-Z]|Foreword|Introduction|Scope|Bibliography|Terms and definitions)\s*$`)
)

// Sections splits raw text into nested sections by heading heuristics.
// If no headings are found, returns a single section with the whole text.
func Sections(raw string) []Section {
	lines := strings.Split(raw, "\n")
	var flat []Section
	cur := -1
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if m := reNumbered.FindStringSubmatch(t); m != nil {
			flat = append(flat, Section{Number: m[1], Title: strings.TrimSpace(m[2])})
			cur = len(flat) - 1
			continue
		}
		if m := reNamed.FindStringSubmatch(t); m != nil {
			flat = append(flat, Section{Number: "", Title: m[1]})
			cur = len(flat) - 1
			continue
		}
		if cur >= 0 {
			if flat[cur].Body != "" {
				flat[cur].Body += "\n"
			}
			flat[cur].Body += line
		}
	}
	for i := range flat {
		flat[i].Body = strings.TrimSpace(flat[i].Body)
	}
	if len(flat) == 0 {
		return []Section{{Body: strings.TrimSpace(raw)}}
	}
	return nest(flat)
}

// nest turns a flat list into a tree by dotted-number depth.
func nest(flat []Section) []Section {
	var roots []Section
	// stack of pointers into roots tree by depth
	type ref struct{ depth int }
	var stack []*Section
	depthOf := func(num string) int {
		if num == "" {
			return 0
		}
		return strings.Count(num, ".")
	}
	add := func(s Section) {
		d := depthOf(s.Number)
		for len(stack) > 0 && depthOf(stack[len(stack)-1].Number) >= d {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, s)
			stack = append(stack, &roots[len(roots)-1])
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, s)
			stack = append(stack, &parent.Children[len(parent.Children)-1])
		}
	}
	for _, s := range flat {
		add(s)
	}
	return roots
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/segment/`
Expected: PASS.

- [ ] **Step 6: Create text fixtures**

`internal/parse/testdata/sample.md`:

```markdown
# Sample Standard

## 1 Scope
This is the scope.

## 4 Context
### 4.1 Understanding
Details here.
```

`internal/parse/testdata/sample.txt`:

```
1 Scope

Plain text scope.

2 References

Some refs.
```

- [ ] **Step 7: Write failing test for text parser**

`internal/parse/text_test.go`:

```go
package parse

import "testing"

func TestParseMarkdown(t *testing.T) {
	doc, err := Parse("testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("no sections")
	}
	flat := doc.Flatten()
	var hasScope, has41 bool
	for _, s := range flat {
		if s.Number == "1" && s.Title == "Scope" {
			hasScope = true
		}
		if s.Number == "4.1" {
			has41 = true
		}
	}
	if !hasScope || !has41 {
		t.Fatalf("missing sections: scope=%v 4.1=%v (%+v)", hasScope, has41, flat)
	}
}

func TestParseText(t *testing.T) {
	doc, err := Parse("testdata/sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("expected sections from numbered text")
	}
}
```

- [ ] **Step 8: Run test to verify it fails**

Run: `go test ./internal/parse/ -run 'TestParseMarkdown|TestParseText'`
Expected: FAIL (undefined `Parse`).

- [ ] **Step 9: Implement `internal/parse/text.go`** (also defines `Parse` dispatch)

```go
package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ringo380/iso-lookup/internal/segment"
)

// Parse reads a local file and returns a normalized Document, dispatching by
// extension. Unknown extensions are treated as plain text.
func Parse(path string) (Document, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return parseHTML(path)
	case ".pdf":
		return parsePDF(path)
	default: // .txt, .md, and anything else
		return parseText(path)
	}
}

var reMDHeading = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
var reMDNumber = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*|[A-Z]\.[0-9]+(?:\.[0-9]+)*)\s+(.+)$`)

func parseText(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	raw := string(b)
	doc := Document{Raw: raw, Title: filepath.Base(path)}

	// Markdown: build sections from # headings, splitting "## 1 Scope" into number+title.
	if strings.ToLower(filepath.Ext(path)) == ".md" && reMDHeading.MatchString(firstHeading(raw)) {
		doc.Sections = sectionsFromMarkdown(raw)
		return doc, nil
	}
	doc.Sections = segment.Sections(raw)
	return doc, nil
}

func firstHeading(raw string) string {
	for _, l := range strings.Split(raw, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

func sectionsFromMarkdown(raw string) []Section {
	lines := strings.Split(raw, "\n")
	var flat []Section
	cur := -1
	for _, line := range lines {
		if m := reMDHeading.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			text := strings.TrimSpace(m[2])
			num, title := "", text
			if nm := reMDNumber.FindStringSubmatch(text); nm != nil {
				num, title = nm[1], strings.TrimSpace(nm[2])
			}
			flat = append(flat, Section{Number: num, Title: title})
			cur = len(flat) - 1
			continue
		}
		if cur >= 0 {
			if flat[cur].Body != "" {
				flat[cur].Body += "\n"
			}
			flat[cur].Body += line
		}
	}
	for i := range flat {
		flat[i].Body = strings.TrimSpace(flat[i].Body)
	}
	if len(flat) == 0 {
		return []Section{{Body: strings.TrimSpace(raw)}}
	}
	return nestByNumber(flat)
}

// nestByNumber nests markdown sections by dotted-number depth (mirrors segment.nest).
func nestByNumber(flat []Section) []Section {
	var roots []Section
	var stack []*Section
	depthOf := func(num string) int {
		if num == "" {
			return 0
		}
		return strings.Count(num, ".")
	}
	for _, s := range flat {
		d := depthOf(s.Number)
		for len(stack) > 0 && depthOf(stack[len(stack)-1].Number) >= d {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, s)
			stack = append(stack, &roots[len(roots)-1])
		} else {
			p := stack[len(stack)-1]
			p.Children = append(p.Children, s)
			stack = append(stack, &p.Children[len(p.Children)-1])
		}
	}
	return roots
}

var _ = fmt.Sprintf // keep fmt import if unused after edits
```

- [ ] **Step 10: Run tests to verify they pass**

Run: `go test ./internal/parse/ -run 'TestParseMarkdown|TestParseText'`
Expected: FAIL to compile — `parseHTML`/`parsePDF` not yet defined. Add temporary stubs to unblock, OR implement Tasks 10–11 before running. Add these stubs to `text.go` temporarily and remove when Tasks 10–11 land:

```go
// temporary stubs (removed in Tasks 10–11)
// func parseHTML(path string) (Document, error) { return parseText(path) }
// func parsePDF(path string) (Document, error) { return parseText(path) }
```

Preferred: implement Tasks 10 and 11 next, then run `go test ./internal/parse/`. Expected after that: PASS.

- [ ] **Step 11: Now build render (Task 8) — it compiles against parse types**

Run: `go test ./internal/render/ ./internal/parse/ ./internal/segment/`
Expected: PASS.

- [ ] **Step 12: Commit**

```bash
git add internal/parse/document.go internal/parse/text.go internal/parse/text_test.go internal/parse/testdata/sample.md internal/parse/testdata/sample.txt internal/segment/
git commit -m "feat(parse,segment): document types, text/markdown parser, heading segmentation"
```

---

## Task 10: HTML parser

**Files:**
- Create: `internal/parse/html.go`, `internal/parse/html_test.go`, `internal/parse/testdata/sample.html`

- [ ] **Step 1: Add dependency**

```bash
go get golang.org/x/net/html@latest
```

- [ ] **Step 2: Create fixture `internal/parse/testdata/sample.html`**

```html
<html><body>
<h1>Sample Standard</h1>
<h2>1 Scope</h2><p>This is the scope.</p>
<h2>4 Context</h2>
<h3>4.1 Understanding</h3><p>Details here.</p>
</body></html>
```

- [ ] **Step 3: Write the failing test**

```go
package parse

import "testing"

func TestParseHTML(t *testing.T) {
	doc, err := Parse("testdata/sample.html")
	if err != nil {
		t.Fatal(err)
	}
	flat := doc.Flatten()
	var hasScope, has41 bool
	for _, s := range flat {
		if s.Number == "1" && s.Title == "Scope" {
			hasScope = true
			if s.Body == "" {
				t.Error("scope body empty")
			}
		}
		if s.Number == "4.1" {
			has41 = true
		}
	}
	if !hasScope || !has41 {
		t.Fatalf("missing sections (%+v)", flat)
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/parse/ -run TestParseHTML`
Expected: FAIL.

- [ ] **Step 5: Implement `internal/parse/html.go`** (remove the temporary `parseHTML` stub from `text.go`)

```go
package parse

import (
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var reHTMLNumber = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*|[A-Z]\.[0-9]+(?:\.[0-9]+)*)\s+(.+)$`)

func parseHTML(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	root, err := html.Parse(strings.NewReader(string(b)))
	if err != nil {
		return Document{}, err
	}

	var flat []Section
	cur := -1
	var title string

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1":
				if title == "" {
					title = strings.TrimSpace(textOf(n))
				}
				return
			case "h2", "h3", "h4", "h5", "h6":
				text := strings.TrimSpace(textOf(n))
				num, ttl := "", text
				if m := reHTMLNumber.FindStringSubmatch(text); m != nil {
					num, ttl = m[1], strings.TrimSpace(m[2])
				}
				flat = append(flat, Section{Number: num, Title: ttl})
				cur = len(flat) - 1
				return
			case "p", "li", "div":
				if cur >= 0 {
					t := strings.TrimSpace(textOf(n))
					if t != "" {
						if flat[cur].Body != "" {
							flat[cur].Body += "\n\n"
						}
						flat[cur].Body += t
					}
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(root)

	doc := Document{Title: title, Raw: textOf(root)}
	if len(flat) == 0 {
		doc.Sections = []Section{{Body: strings.TrimSpace(doc.Raw)}}
	} else {
		doc.Sections = nestByNumber(flat)
	}
	return doc, nil
}

func textOf(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			sb.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(sb.String()), " ")
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/parse/ -run TestParseHTML`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/parse/html.go internal/parse/html_test.go internal/parse/testdata/sample.html go.mod go.sum
git commit -m "feat(parse): HTML parser via x/net/html"
```

---

## Task 11: PDF parser

**Files:**
- Create: `internal/parse/pdf.go`, `internal/parse/pdf_test.go`

- [ ] **Step 1: Add dependency**

```bash
go get github.com/ledongthuc/pdf@latest
```

- [ ] **Step 2: Write the failing test (tolerant: extraction-or-graceful)**

```go
package parse

import (
	"os"
	"testing"
)

func TestParsePDFGraceful(t *testing.T) {
	// Use any small PDF present in testdata; if none, skip.
	const p = "testdata/sample.pdf"
	if _, err := os.Stat(p); err != nil {
		t.Skip("no sample.pdf fixture; provide one to exercise PDF extraction")
	}
	doc, err := Parse(p)
	if err != nil {
		t.Fatalf("parse pdf: %v", err)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("expected at least a fallback section")
	}
}
```

- [ ] **Step 3: Run test to verify it fails/skips**

Run: `go test ./internal/parse/ -run TestParsePDFGraceful`
Expected: FAIL to compile (undefined `parsePDF`), or SKIP after implementation if no fixture.

- [ ] **Step 4: Implement `internal/parse/pdf.go`** (remove the temporary `parsePDF` stub from `text.go`)

```go
package parse

import (
	"bytes"
	"strings"

	"github.com/ledongthuc/pdf"

	"github.com/ringo380/iso-lookup/internal/segment"
)

func parsePDF(path string) (Document, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()

	var buf bytes.Buffer
	if rd, err := r.GetPlainText(); err == nil {
		buf.ReadFrom(rd)
	}
	raw := buf.String()

	doc := Document{Raw: raw}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		// image-only / unextractable PDF: graceful fallback.
		doc.Sections = []Section{{
			Title: "Full text not extractable",
			Body:  "This PDF appears to be image-only or could not be parsed. Use `iso open` for the official page.",
		}}
		return doc, nil
	}
	doc.Sections = segment.Sections(raw)
	return doc, nil
}
```

- [ ] **Step 5: Run tests for the whole parse package**

Run: `go test ./internal/parse/`
Expected: PASS (PDF test SKIPs without a fixture; others pass). Ensure the temporary stubs in `text.go` are removed.

- [ ] **Step 6: Commit**

```bash
git add internal/parse/pdf.go internal/parse/pdf_test.go go.mod go.sum
git commit -m "feat(parse): PDF text extraction with image-only fallback"
```

---

## Task 12: library (local-file discovery)

**Files:**
- Create: `internal/library/library.go`, `internal/library/library_test.go`

- [ ] **Step 1: Add YAML dependency**

```bash
go get gopkg.in/yaml.v3@latest
```

- [ ] **Step 2: Write the failing test**

```go
package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindByConvention(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ISO-IEC-27001-2022.pdf"), []byte("x"), 0o644)
	lib := New(dir, "")
	got, ok := lib.Find("ISO/IEC 27001:2022")
	if !ok {
		t.Fatal("convention match failed")
	}
	if filepath.Base(got) != "ISO-IEC-27001-2022.pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestFindByConventionBareNumber(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "iso27001.md"), []byte("x"), 0o644)
	lib := New(dir, "")
	if _, ok := lib.Find("27001"); !ok {
		t.Fatal("bare-number convention match failed")
	}
}

func TestIndexOverridesConvention(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ISO-IEC-27001-2022.pdf"), []byte("x"), 0o644)
	custom := filepath.Join(dir, "my-copy.txt")
	os.WriteFile(custom, []byte("x"), 0o644)
	idx := filepath.Join(dir, "index.yaml")
	os.WriteFile(idx, []byte("entries:\n  \"ISO/IEC 27001:2022\": my-copy.txt\n"), 0o644)
	lib := New(dir, idx)
	got, ok := lib.Find("ISO/IEC 27001:2022")
	if !ok || filepath.Base(got) != "my-copy.txt" {
		t.Fatalf("index override failed: %q ok=%v", got, ok)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/library/`
Expected: FAIL (undefined).

- [ ] **Step 4: Implement `library.go`**

```go
package library

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Library resolves an ISO reference to a local file path.
type Library struct {
	docsDir   string
	indexFile string
}

// New constructs a Library. indexFile may be empty.
func New(docsDir, indexFile string) *Library {
	return &Library{docsDir: docsDir, indexFile: indexFile}
}

type indexFileShape struct {
	Entries map[string]string `yaml:"entries"`
}

var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func canon(s string) string {
	return reNonAlnum.ReplaceAllString(strings.ToLower(s), "")
}

// Find returns the local file path for ref, index override first.
func (l *Library) Find(ref string) (string, bool) {
	if l.docsDir == "" {
		return "", false
	}
	// 1. index override
	if l.indexFile != "" {
		if b, err := os.ReadFile(l.indexFile); err == nil {
			var idx indexFileShape
			if yaml.Unmarshal(b, &idx) == nil {
				for k, v := range idx.Entries {
					if canon(k) == canon(ref) {
						if filepath.IsAbs(v) {
							return v, true
						}
						return filepath.Join(l.docsDir, v), true
					}
				}
			}
		}
	}
	// 2. filename convention
	want := canon(ref)
	entries, err := os.ReadDir(l.docsDir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if canon(stem) == want || strings.Contains(want, canon(stem)) || strings.Contains(canon(stem), want) {
			return filepath.Join(l.docsDir, name), true
		}
	}
	return "", false
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/library/`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
git add internal/library/ go.mod go.sum
git commit -m "feat(library): local-file discovery by convention + index override"
```

---

## Task 13: shared catalog loader + `iso search` + `iso show`

**Files:**
- Create: `cmd/iso/loader.go`, `cmd/iso/cmd_search.go`, `cmd/iso/cmd_show.go`

- [ ] **Step 1: Implement `cmd/iso/loader.go` (shared helpers)**

```go
package main

import (
	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/ringo380/iso-lookup/internal/library"
)

func loadCatalog() (*catalog.Catalog, error) {
	path, err := config.CachePath()
	if err != nil {
		return nil, err
	}
	recs, err := catalog.LoadIndex(path)
	if err != nil {
		return nil, err
	}
	return catalog.New(recs), nil
}

func loadLibrary() (*library.Library, error) {
	c, err := config.Load()
	if err != nil {
		return nil, err
	}
	return library.New(c.DocsDir, c.IndexFile), nil
}
```

- [ ] **Step 2: Implement `cmd/iso/cmd_search.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var searchJSON bool

var searchCmd = &cobra.Command{
	Use:   "search <terms...>",
	Short: "Search standards by keyword",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		res := c.Search(strings.Join(args, " "))
		if searchJSON {
			return json.NewEncoder(os.Stdout).Encode(res)
		}
		if len(res) > 50 {
			res = res[:50]
			fmt.Fprintln(os.Stderr, "(showing first 50 matches; refine your query)")
		}
		fmt.Print(render.SearchList(res))
		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output JSON")
	rootCmd.AddCommand(searchCmd)
}
```

- [ ] **Step 3: Implement `cmd/iso/cmd_show.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ringo380/iso-lookup/internal/parse"
	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var (
	showJSON        bool
	showInteractive bool
)

var showCmd = &cobra.Command{
	Use:   "show <reference>",
	Short: "Show metadata up front, plus a table of contents if a local file exists",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(args[0])
		if !ok {
			fmt.Fprintf(os.Stderr, "no exact match for %q; closest:\n", args[0])
			fmt.Print(render.SearchList(limit(c.Search(args[0]), 10)))
			return fmt.Errorf("not found")
		}
		if showJSON {
			return json.NewEncoder(os.Stdout).Encode(rec)
		}
		fmt.Print(render.Summary(rec))

		lib, err := loadLibrary()
		if err != nil {
			return err
		}
		path, ok := lib.Find(rec.Reference)
		if !ok {
			fmt.Print(render.NoLocalFile(rec))
			return nil
		}
		doc, err := parse.Parse(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", path, err)
			return nil
		}
		if showInteractive {
			return runTUI(rec, doc)
		}
		fmt.Print(render.TOC(doc))
		return nil
	},
}

func limit(recs []parse.Record, n int) []parse.Record { return recs } // placeholder removed below

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "output JSON")
	showCmd.Flags().BoolVar(&showInteractive, "interactive", false, "browse in the TUI")
	rootCmd.AddCommand(showCmd)
}
```

Replace the broken `limit` helper (wrong type) with this correct version in `cmd/iso/loader.go`:

```go
import "github.com/ringo380/iso-lookup/internal/catalog"

func limit(recs []catalog.Record, n int) []catalog.Record {
	if len(recs) > n {
		return recs[:n]
	}
	return recs
}
```

And delete the placeholder `limit` line from `cmd_show.go`. (`runTUI` is defined in Task 16; add a temporary stub in `loader.go`: `func runTUI(catalog.Record, parse.Document) error { return nil }` and replace it in Task 16.)

- [ ] **Step 4: Build and smoke-test against the real index**

Run:
```bash
go build -o iso ./cmd/iso
./iso update            # if not already built
./iso search information security management
./iso show "ISO/IEC 27001:2022"
./iso show 27001
```
Expected: search lists matches; show prints the 27001 summary block (title, Published, committee, ICS, scope, URL) and then the no-local-file notice (until a local file is added).

- [ ] **Step 5: Commit**

```bash
git add cmd/iso/loader.go cmd/iso/cmd_search.go cmd/iso/cmd_show.go
git commit -m "feat(cli): search and show commands"
```

---

## Task 14: `iso chapter` command

**Files:**
- Create: `cmd/iso/cmd_chapter.go`

- [ ] **Step 1: Implement `cmd_chapter.go`**

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/ringo380/iso-lookup/internal/parse"
	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var noPager bool

var chapterCmd = &cobra.Command{
	Use:   "chapter <reference> <section>",
	Short: "Print a single chapter/segment from the local file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, want := args[0], args[1]
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(ref)
		if !ok {
			return fmt.Errorf("no match for %q", ref)
		}
		lib, err := loadLibrary()
		if err != nil {
			return err
		}
		path, ok := lib.Find(rec.Reference)
		if !ok {
			return fmt.Errorf("no local file for %s; run `iso open %s`", rec.Reference, rec.Reference)
		}
		doc, err := parse.Parse(path)
		if err != nil {
			return err
		}
		for _, s := range doc.Flatten() {
			if strings.EqualFold(s.Number, want) || strings.EqualFold(s.Title, want) {
				out := render.Chapter(s)
				return page(out)
			}
		}
		return fmt.Errorf("section %q not found; run `iso show %s` for the contents", want, rec.Reference)
	},
}

func page(text string) error {
	cfg, _ := config.Load()
	pager := cfg.Pager
	if noPager || pager == "" {
		fmt.Print(text)
		return nil
	}
	c := exec.Command(pager)
	c.Stdin = strings.NewReader(text)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func init() {
	chapterCmd.Flags().BoolVar(&noPager, "no-pager", false, "do not pipe through a pager")
	rootCmd.AddCommand(chapterCmd)
}
```

- [ ] **Step 2: Smoke-test with a local fixture**

Run:
```bash
mkdir -p /tmp/isodocs && cp internal/parse/testdata/sample.md /tmp/isodocs/ISO-IEC-27001-2022.md
go build -o iso ./cmd/iso
./iso config set-docs /tmp/isodocs   # implemented in Task 17; until then set config.json manually
./iso chapter 27001 1
```
Expected: prints clause "1 Scope" body. (If Task 17 not done yet, hand-write `~/.config/iso-lookup/config.json` with `{"docs_dir":"/tmp/isodocs"}`.)

- [ ] **Step 3: Commit**

```bash
git add cmd/iso/cmd_chapter.go
git commit -m "feat(cli): chapter command with pager support"
```

---

## Task 15: `iso open` command

**Files:**
- Create: `cmd/iso/cmd_open.go`, `cmd/iso/open_test.go`

- [ ] **Step 1: Write the failing test for the platform command builder**

```go
package main

import "testing"

func TestOpenArgs(t *testing.T) {
	bin, args := openCommand("darwin", "https://example.com")
	if bin != "open" || len(args) != 1 || args[0] != "https://example.com" {
		t.Fatalf("darwin: %s %v", bin, args)
	}
	bin, _ = openCommand("linux", "https://example.com")
	if bin != "xdg-open" {
		t.Fatalf("linux bin %s", bin)
	}
	bin, args = openCommand("windows", "https://example.com")
	if bin != "rundll32" || args[0] != "url.dll,FileProtocolHandler" {
		t.Fatalf("windows: %s %v", bin, args)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/iso/ -run TestOpenArgs`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `cmd_open.go`**

```go
package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func openCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

var openCmd = &cobra.Command{
	Use:   "open <reference>",
	Short: "Open the official ISO URL in your browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(args[0])
		if !ok {
			return fmt.Errorf("no match for %q", args[0])
		}
		bin, cargs := openCommand(runtime.GOOS, rec.URL)
		fmt.Println("Opening", rec.URL)
		return exec.Command(bin, cargs...).Start()
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/iso/ -run TestOpenArgs`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/iso/cmd_open.go cmd/iso/open_test.go
git commit -m "feat(cli): open command (cross-platform browser launch)"
```

---

## Task 16: TUI browser + `iso browse`

**Files:**
- Create: `internal/tui/tui.go`, `cmd/iso/cmd_browse.go`
- Modify: `cmd/iso/loader.go` (replace `runTUI` stub)

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Implement `internal/tui/tui.go`**

```go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/parse"
)

type model struct {
	rec      catalog.Record
	sections []parse.Section
	cursor   int
	scroll   int
	width    int
	height   int
}

// flat is a display row: a section with an indent depth.
type flat struct {
	sec   parse.Section
	depth int
}

func flatten(secs []parse.Section, depth int, out *[]flat) {
	for _, s := range secs {
		*out = append(*out, flat{s, depth})
		flatten(s.Children, depth+1, out)
	}
}

// New builds the TUI model for a record + parsed document.
func New(rec catalog.Record, doc parse.Document) model {
	var rows []flat
	flatten(doc.Sections, 0, &rows)
	flatSecs := make([]parse.Section, len(rows))
	for i, r := range rows {
		s := r.sec
		s.Title = strings.Repeat("  ", r.depth) + s.Title
		flatSecs[i] = s
	}
	return model{rec: rec, sections: flatSecs}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.sections)-1 {
				m.cursor++
				m.scroll = 0
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.scroll = 0
			}
		}
	}
	return m, nil
}

var (
	listStyle = lipgloss.NewStyle().Width(34).Border(lipgloss.NormalBorder(), false, true, false, false).Padding(0, 1)
	selStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	bodyStyle = lipgloss.NewStyle().Padding(0, 2)
)

func (m model) View() string {
	if len(m.sections) == 0 {
		return "No sections. Press q to quit."
	}
	var left strings.Builder
	for i, s := range m.sections {
		line := strings.TrimSpace(s.Number + " " + s.Title)
		if i == m.cursor {
			line = selStyle.Render("> " + line)
		} else {
			line = "  " + line
		}
		left.WriteString(line + "\n")
	}
	cur := m.sections[m.cursor]
	body := fmt.Sprintf("%s  %s\n\n%s", cur.Number, strings.TrimSpace(cur.Title), cur.Body)
	footer := "\n\n[↑/↓ navigate · q quit] " + m.rec.Reference + "  " + m.rec.URL
	right := bodyStyle.Render(body + footer)
	return lipgloss.JoinHorizontal(lipgloss.Top, listStyle.Render(left.String()), right)
}

// Run launches the interactive browser.
func Run(rec catalog.Record, doc parse.Document) error {
	_, err := tea.NewProgram(New(rec, doc), tea.WithAltScreen()).Run()
	return err
}
```

- [ ] **Step 3: Replace the `runTUI` stub in `cmd/iso/loader.go`**

```go
import (
	"github.com/ringo380/iso-lookup/internal/parse"
	"github.com/ringo380/iso-lookup/internal/tui"
)

func runTUI(rec catalog.Record, doc parse.Document) error {
	return tui.Run(rec, doc)
}
```

- [ ] **Step 4: Implement `cmd/iso/cmd_browse.go`**

```go
package main

import (
	"fmt"

	"github.com/ringo380/iso-lookup/internal/parse"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse <reference>",
	Short: "Browse a standard interactively (TUI)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(args[0])
		if !ok {
			return fmt.Errorf("no match for %q", args[0])
		}
		lib, err := loadLibrary()
		if err != nil {
			return err
		}
		path, ok := lib.Find(rec.Reference)
		if !ok {
			return fmt.Errorf("no local file for %s; the TUI needs full text. Run `iso open %s`", rec.Reference, rec.Reference)
		}
		doc, err := parse.Parse(path)
		if err != nil {
			return err
		}
		return runTUI(rec, doc)
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
}
```

- [ ] **Step 5: Build and verify it compiles + TUI launches**

Run: `go build -o iso ./cmd/iso && ./iso browse 27001` (with a local file mapped)
Expected: full-screen list+body view; ↑/↓ navigates; q quits. Without a local file: clear error pointing to `iso open`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/ cmd/iso/cmd_browse.go cmd/iso/loader.go go.mod go.sum
git commit -m "feat(tui): interactive browser and browse command"
```

---

## Task 17: `iso config` command

**Files:**
- Create: `cmd/iso/cmd_config.go`

- [ ] **Step 1: Implement `cmd_config.go`**

```go
package main

import (
	"fmt"

	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or set configuration (docs folder, index file, pager)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("docs_dir:   %s\nindex_file: %s\npager:      %s\n", c.DocsDir, c.IndexFile, c.Pager)
		return nil
	},
}

var configSetDocs = &cobra.Command{
	Use:   "set-docs <path>",
	Short: "Set the local docs folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _ := config.Load()
		c.DocsDir = args[0]
		return config.Save(c)
	},
}

var configSetIndex = &cobra.Command{
	Use:   "set-index <path>",
	Short: "Set the optional index.yaml override file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _ := config.Load()
		c.IndexFile = args[0]
		return config.Save(c)
	},
}

var configSetPager = &cobra.Command{
	Use:   "set-pager <command>",
	Short: "Set the pager (e.g. less); empty disables paging",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _ := config.Load()
		c.Pager = args[0]
		return config.Save(c)
	},
}

func init() {
	configCmd.AddCommand(configSetDocs, configSetIndex, configSetPager)
	rootCmd.AddCommand(configCmd)
}
```

- [ ] **Step 2: Build and verify**

Run:
```bash
go build -o iso ./cmd/iso
./iso config set-docs /tmp/isodocs
./iso config
```
Expected: prints docs_dir = /tmp/isodocs.

- [ ] **Step 3: Commit**

```bash
git add cmd/iso/cmd_config.go
git commit -m "feat(cli): config command"
```

---

## Task 18: end-to-end verification + README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: all packages PASS (PDF test SKIPs without a fixture).

- [ ] **Step 2: Run `go vet` and build**

Run: `go vet ./... && go build -o iso ./cmd/iso`
Expected: no vet errors, clean build.

- [ ] **Step 3: Full manual flow against real data**

Run:
```bash
./iso update
./iso search information security management
./iso show "ISO/IEC 27001:2022"
mkdir -p /tmp/isodocs && cp internal/parse/testdata/sample.md /tmp/isodocs/ISO-IEC-27001-2022.md
./iso config set-docs /tmp/isodocs
./iso show 27001        # now shows TOC
./iso chapter 27001 1
./iso open 27001
```
Expected: each command behaves per the spec; `show` adds a TOC once the local file is present.

- [ ] **Step 4: Write `README.md`**

```markdown
# iso

A CLI for looking up ISO security and standards documents.

- **Metadata** for every standard comes from the official [ISO Open Data](https://www.iso.org/open-data.html) dataset (ODC-By 1.0 — data © ISO), cached locally and queried offline.
- **Full text** is read from local files you provide (PDF / text / Markdown / HTML); ISO standard bodies are copyrighted and not freely distributable.

## Install

```bash
go build -o iso ./cmd/iso
```

## Usage

```bash
iso update                       # download/refresh the metadata index (run first)
iso search <terms...>            # keyword search
iso show <reference>             # metadata summary + table of contents
iso chapter <reference> <sec>    # print one chapter from a local file
iso open <reference>             # open the official ISO page
iso browse <reference>           # interactive TUI
iso config set-docs <path>       # point at your local standards folder
```

Local files are matched by filename (e.g. `ISO-IEC-27001-2022.pdf`) or via an
optional `index.yaml` in the docs folder:

```yaml
entries:
  "ISO/IEC 27001:2022": my-purchased-copy.pdf
```

## Attribution

Standards metadata © ISO, via the ISO Open Data initiative, licensed under ODC-By 1.0.
```

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: add README with usage and attribution"
```

- [ ] **Step 6: Archive the plan**

```bash
mkdir -p docs/superpowers/plans/archive
git mv docs/superpowers/plans/2026-06-01-iso-lookup.md docs/superpowers/plans/archive/ 2>/dev/null || true
git add -A && git commit -m "chore: archive iso-lookup plan"
```

---

## Self-Review notes

- **Spec coverage:** search ✓ (T13), show-with-summary ✓ (T13), chapter ✓ (T14), open ✓ (T15), browse/TUI ✓ (T16), update/Open Data ingest ✓ (T4–7), local discovery convention+index ✓ (T12), PDF/text/MD/HTML parse ✓ (T9–11), segmentation ✓ (T9), stage-code map ✓ (T2), committee/ICS resolution ✓ (T4), HTML scope strip ✓ (T3), config ✓ (T1,T17), error handling (no index / no file / image-only PDF / not found) ✓, attribution ✓ (T7,T18), `--json`/`--no-pager`/`--interactive` ✓.
- **Build-order caveat:** `render` (T8) and `cmd` packages depend on `parse` types and the `runTUI` stub. The plan calls these out explicitly (T9 step 1 defines types first; T13 adds the `runTUI` stub; T16 replaces it). Follow task order.
- **Type consistency:** `Record`, `Document`, `Section`, `Catalog.Lookup/Search`, `Library.Find`, `parse.Parse`, `segment.Sections`, `StageLabel`, `StripHTML`, `Ingest`, `BuildIndex`, `SaveIndex`/`LoadIndex` names are used consistently across tasks. `limit` helper corrected to `catalog.Record` in T13.
```
