# iso

A CLI for looking up ISO security and standards documents.

- **Metadata** for every standard comes from the official [ISO Open Data](https://www.iso.org/open-data.html) dataset (ODC-By 1.0 — data © ISO), cached locally and queried offline.
- **Full text** is read from local files you provide (PDF / text / Markdown / HTML); ISO standard bodies are copyrighted and not freely distributable.

## Install

**Homebrew (macOS):**

```bash
brew install robworks-code/tap/iso
```

**Go (any platform, needs Go 1.25+):**

```bash
go install github.com/Robworks-Code/iso-lookup/cmd/iso@latest
```

Prebuilt binaries, building from source, and verification are covered in
[INSTALL.md](INSTALL.md). Release notes are in [CHANGELOG.md](CHANGELOG.md).

## Usage

```bash
iso update                       # download/refresh the metadata index (run first)
iso search <terms...>            # keyword search
iso show <reference>             # metadata summary + table of contents
iso chapter <reference> <sec>    # print one chapter from a local file
iso open <reference>             # open the official ISO page
iso browse <reference>           # interactive TUI
iso scan [path]                  # detect a project's stack, recommend standards
iso config set-docs <path>       # point at your local standards folder
```

References can be exact (`ISO/IEC 27001:2022`) or a bare number (`27001`); bare
numbers resolve to the current published standard.

Local files are matched by filename (e.g. `ISO-IEC-27001-2022.pdf`) or via an
optional `index.yaml` in the docs folder:

```yaml
entries:
  "ISO/IEC 27001:2022": my-purchased-copy.pdf
```

Point `iso` at your index file with `iso config set-index <path>`.

## Color output

Listings are color-coded for readability: references stand out, statuses are
colored by state (green published, red withdrawn, yellow under review), and
group headers, rationale, and evidence are visually distinct.

Color is automatic - it turns off when output is piped or redirected, and honors
the [`NO_COLOR`](https://no-color.org) convention. Two global flags override the
auto-detection:

| Flag | Effect |
|------|--------|
| `--no-color` | Disable color (also via `NO_COLOR=1`). |
| `--color` | Force color even when piped (e.g. `iso scan . --color \| less -R`). |

## Scanning a project

`iso scan` inspects a folder, detects its technology stack from marker files and
dependency manifests, and recommends the current ISO standards relevant to what
it finds — grouped into a report you can reshape.

```bash
iso scan .                       # detect stack + recommend standards (current dir)
iso scan ./service               # scan another folder
iso scan . --interactive         # browse the report in an interactive TUI
iso scan stack .                 # just the detected components, no recommendations
iso scan why security .          # explain what drove a component/category/concern
```

Detection reads markers like `go.mod`, `package.json`, `Dockerfile`, `*.tf`, and
CI configs (and, where present, their dependency lists) and maps them to concerns
such as information security, privacy, AI, software lifecycle, and accessibility.
Each concern resolves to a curated set of anchor standards from the offline
index; `--discover` broadens the set via catalog search. ISO standards address
domains, not specific products, so recommendations are advisory starting points,
not a compliance checklist.

| Flag | Effect |
|------|--------|
| `--group-by component\|category\|ics` | How to group standards (default: `component`). |
| `--category <name>` | Keep only groups/standards matching a category. |
| `--component <name>` | Keep only standards driven by a matching component. |
| `--discover` | Add related standards via catalog search (lower confidence). |
| `--include-drafts` | Include drafts and withdrawn standards (default: current only). |
| `--limit <n>` | Cap standards per group (0 = no limit). |
| `--depth <n>` | Maximum directory depth to scan (default 6; 0 = unlimited). |
| `--sort <key>` | Order within each group: relevance, reference, date, status. |
| `--long`, `-l` | Wide listing with publication date and committee. |
| `--interactive`, `-i` | Browse the grouped report in a two-pane TUI (groups left, standards + rationale right; arrows/`j`/`k` to switch, `f`/`b` to scroll, `q` to quit). |
| `--json` | Machine-readable output. |

## Attribution

Standards metadata © ISO, via the ISO Open Data initiative, licensed under ODC-By 1.0.
