# iso-lookup — Design Spec

**Created**: 2026-06-01
**Status**: In Progress

## Overview

`iso` is a Go CLI for looking up security and standards documents by keyword or
by direct ISO reference. It prints authoritative metadata (title, scope, status,
official URL) up front, and — when a local copy of the standard is available —
lets the user read and navigate the document chapter-by-chapter or open the
official URL directly.

### The data-source constraint (why the tool is split in two)

Full ISO standard text is **not** freely available — standards are copyrighted
and sold per-document. There is no public API for body text. Therefore the tool
draws from two distinct sources:

1. **Metadata** — the official, free **ISO Open Data** dataset (ODC-By 1.0
   license). Machine-readable, authoritative, covers the entire catalogue
   (~80,726 deliverables). Used for search and the "basic content up front"
   summary. Works fully offline after a one-time download.
2. **Full text** — **local files the user provides** (PDF / plain text /
   Markdown / HTML). Used for chapter navigation and reading. The tool never
   pretends to have body text it doesn't have.

This split is deliberate and load-bearing: the summary experience works for
*every* standard out of the box; full-text reading works for whatever the user
has on disk.

## Goals

- Look up a standard by direct reference (`ISO/IEC 27001:2022`, or loose forms
  like `27001`) or by keyword search.
- Print a useful summary up front: title, edition, publication date, status,
  scope, owning committee, ICS classification, official URL.
- When a local file is mapped to the standard, show a table of contents and let
  the user read any chapter/segment.
- Open the official ISO URL in the browser.
- Default to plain, scriptable stdout; offer an interactive TUI for browsing.
- Work offline after the metadata index is built.

## Non-Goals (YAGNI)

- Scraping iso.org web pages (Open Data replaces this entirely).
- Fetching/storing copyrighted full text from any remote source.
- Editing or annotating standards.
- A daemon, server, or plugin architecture.
- OCR of scanned/image-only PDFs (v1 handles text-bearing PDFs only; image-only
  PDFs degrade gracefully — see Error Handling).

## Architecture

Layered packages behind interfaces, so the two hard problems (metadata ingest,
local-file parsing/segmentation) are independently testable and the catalog
source can change without touching the rest.

```
cmd/iso/                  main + cobra command wiring
internal/
  catalog/                ISO Open Data ingest + offline query
  library/                local-file discovery (convention + index override)
  parse/                  per-format parsers -> normalized Document
  segment/                heading/numbering heuristics -> []Section
  render/                 stdout rendering + pager
  tui/                    Bubble Tea interactive browser
  config/                 paths, config file, cache location
```

### Dependencies (kept lean)

- `github.com/spf13/cobra` — command structure.
- `github.com/charmbracelet/bubbletea` + `lipgloss` — TUI.
- A pure-Go PDF text extractor (e.g. `github.com/ledongthuc/pdf`); evaluate at
  implementation time, fall back to `pdftotext` shell-out if quality is poor.
- `golang.org/x/net/html` — HTML parsing and scope-tag stripping.
- Standard library for JSONL streaming, in-memory search, gob cache.
- No Parquet dependency (JSONL is parsed once, then discarded).

## Components

### catalog — ISO Open Data ingest + query

**Responsibility:** provide authoritative metadata for any ISO standard,
offline.

- **Source files** (Azure blob, `_latest`):
  - `iso_deliverables_metadata/json/iso_deliverables_metadata.jsonl`
  - `iso_technical_committees/json/iso_technical_committees.jsonl`
  - `iso_ics/csv/ICS.csv`
- **Build step (`iso update`):** download JSONL, parse, project to a slim record
  (fields below), resolve committee + ICS names from the companion datasets,
  strip HTML from scope, and write a compact local index (gob) under the cache
  dir. JSONL is not retained after the index is built.
- **Slim record:**
  ```
  Reference     string   // "ISO/IEC 27001:2022" — primary key
  Title         string   // title.en
  Scope         string   // scope.en, HTML stripped to plain text
  Edition       int
  PublishedDate string   // publicationDate
  StageCode     int      // currentStage
  Status        string   // human-readable, from stage-code map
  ICS           []string // codes + resolved names
  Committee     string   // ownerCommittee + resolved name
  Replaces      string
  ReplacedBy    string
  Pages         int
  ID            int      // -> official URL
  URL           string   // derived: https://www.iso.org/standard/{ID}.html
  ```
- **Interface:**
  ```go
  type Catalog interface {
      Lookup(ref string) (Record, bool) // exact + normalized matching
      Search(terms string) []Record     // token/substring over ref+title+scope
  }
  ```
- **Lookup normalization:** accept `27001`, `iso 27001`, `ISO/IEC 27001:2022`,
  case-insensitive; if multiple editions match a bare number, prefer the
  non-replaced (current) one and note the others.
- **Search:** in-memory token match over reference + title + scope, ranked
  (reference match > title match > scope match). 80k records → trivial.
- **Stage-code map:** small static table of ISO Harmonized Stage Codes
  (e.g. 6060 → "Published", 9599 → "Withdrawn", etc.). Unknown codes fall back
  to printing the raw code.

### library — local-file discovery

**Responsibility:** map an ISO reference to a local file, if one exists.

- **Convention:** scan a configured docs folder for filenames encoding the
  reference, e.g. `ISO-IEC-27001-2022.pdf`, `ISO_27001_2022.md`,
  `iso27001.txt`. Normalize filename → candidate reference.
- **Index override:** optional `index.yaml` in the docs folder mapping explicit
  references → paths (+ optional title/URL overrides) for messy filenames the
  convention misses.
- **Resolution order:** index entry wins over convention match.
- **Interface:**
  ```go
  type Library interface {
      Find(ref string) (path string, ok bool)
  }
  ```

### parse — per-format parsers

**Responsibility:** turn a local file into a normalized in-memory document.

- One parser per format, dispatched by extension/content sniff:
  - `.txt`, `.md` — line-based.
  - `.html`, `.htm` — DOM headings via `x/net/html`.
  - `.pdf` — text extraction; structure inferred from extracted text.
- **Normalized type:**
  ```go
  type Document struct {
      Title    string
      Sections []Section
      Raw      string // full extracted text, for fallback
  }
  type Section struct {
      Number   string // "4.1", "A.5", "Annex A"
      Title    string
      Body     string
      Children []Section
  }
  ```

### segment — heading/numbering heuristics

**Responsibility:** detect chapter/segment boundaries from parsed text.

- Recognize numbered headings (`1`, `4.1`, `4.1.2`), lettered annex clauses
  (`A.5`, `A.5.1`), and `Annex A`/`Bibliography`/`Foreword`/`Scope` markers.
- Markdown/HTML: use native heading levels first; fall back to numbering.
- Output a nested `[]Section`. If no structure is detected, produce a single
  section containing the raw text.

### render — stdout output

**Responsibility:** format output for the terminal.

- `show`: summary block (title, ref, status, dates, committee, ICS, URL, scope)
  followed by a numbered TOC when a local file is present, or a
  "full text not available locally — `iso open` for the official page" notice
  when not.
- `chapter`: print one section's body, optionally piped through `$PAGER`.
- `search`: compact list (reference · title · status).

### tui — interactive browser

**Responsibility:** full-screen browsing for one standard.

- Left pane: section list (TOC). Right pane: scrollable section body.
- Keys: navigate sections, scroll, `o` to open URL, `q` to quit.
- Launched by `iso browse <ref>` or any command with `--interactive`.

## Commands

```
iso search <terms...>          # keyword search -> list of matches
iso show <ref>                 # summary up front + TOC (core command)
iso chapter <ref> <section>    # print one segment, e.g. "iso chapter 27001 A.5"
iso open <ref>                 # open official ISO URL in browser
iso browse <ref>               # interactive TUI (also: <cmd> --interactive)
iso update                     # download/refresh the ISO Open Data index
iso config                     # show/set docs folder + index path
```

Global flags: `--interactive`, `--no-pager`, `--json` (machine-readable output
for `search`/`show`).

## Data Flow

```
command
  -> resolve ref (catalog.Lookup, or catalog.Search for `search`)
  -> catalog provides Record (offline, from local index)
  -> library.Find(ref) -> path?
       yes -> parse(path) -> segment -> Document
       no  -> metadata-only
  -> render (stdout) or tui (interactive)
```

`iso update` is a separate one-time/occasional flow: download datasets → build
slim index → write gob cache.

## Error Handling

- **No index yet:** any query that needs the catalog prints a clear
  "run `iso update` first" message and exits non-zero.
- **Network failure during `update`:** keep the existing cached index untouched;
  report the failure.
- **Reference not found:** suggest closest matches (search fallback).
- **No local file:** show metadata + URL, explicitly note full text isn't local.
- **Parser failure / image-only PDF:** fall back to `Document` with a single
  raw-text section; if extraction yields nothing, report that the file appears
  to be image-only/unparseable and still offer `iso open`.
- **Unknown stage code / missing committee name:** degrade to raw value rather
  than failing.

## Configuration

- Config + cache under `~/.config/iso-lookup/` (respect `XDG_CONFIG_HOME`).
  - `config.yaml` — docs folder path, index file path, pager preference.
  - `catalog.gob` — built metadata index.
- `iso config` reads/writes `config.yaml`.

## Testing

- **catalog:** ingest against a small recorded JSONL fixture; verify field
  projection, HTML stripping, stage-code mapping, lookup normalization, search
  ranking.
- **library:** temp dirs exercising filename convention + `index.yaml` override
  precedence.
- **parse:** fixture files per format (txt, md, html, a small text PDF).
- **segment:** unit tests for numbered/annex/heading detection and the
  no-structure fallback.
- **render:** golden-file tests for summary, TOC, chapter, and the
  no-local-file notice.
- HTTP downloads in `update` go through an injectable client (httptest).

## Open Implementation Questions (resolve early, low risk)

- Confirm `id` → `iso.org/standard/{id}.html` URL mapping against a few real
  records before relying on it; fall back to a search URL if it doesn't hold.
- Choose the PDF library vs. `pdftotext` shell-out based on extraction quality
  on a real purchased standard.
- Finalize the ISO Harmonized Stage Code → label table (cover the common codes;
  raw fallback for the rest).

## Attribution

Per ODC-By 1.0, surface ISO attribution for the Open Data (e.g. in `iso update`
output and a `--version`/about footer): data © ISO, via the ISO Open Data
initiative, licensed under ODC-By 1.0.
