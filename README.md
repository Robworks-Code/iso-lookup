# iso

A CLI for looking up ISO security and standards documents.

- **Metadata** for every standard comes from the official [ISO Open Data](https://www.iso.org/open-data.html) dataset (ODC-By 1.0 — data © ISO), cached locally and queried offline.
- **Full text** is read from local files you provide (PDF / text / Markdown / HTML); ISO standard bodies are copyrighted and not freely distributable.

## Install

**Homebrew (macOS):**

```bash
brew install Robworks-Code/tap/iso
```

**Go (any platform, needs Go 1.25+):**

```bash
go install github.com/Robworks-Code/iso-lookup/cmd/iso@latest
```

**Prebuilt binaries:** download a tarball for your OS/arch from the
[releases page](https://github.com/Robworks-Code/iso-lookup/releases) and put
`iso` on your `PATH`.

**From source:**

```bash
make install            # builds and installs into /usr/local/bin (override PREFIX=...)
# or
make build              # builds ./bin/iso
go install ./cmd/iso    # installs into $(go env GOPATH)/bin
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

References can be exact (`ISO/IEC 27001:2022`) or a bare number (`27001`); bare
numbers resolve to the current published standard.

Local files are matched by filename (e.g. `ISO-IEC-27001-2022.pdf`) or via an
optional `index.yaml` in the docs folder:

```yaml
entries:
  "ISO/IEC 27001:2022": my-purchased-copy.pdf
```

Point `iso` at your index file with `iso config set-index <path>`.

## Attribution

Standards metadata © ISO, via the ISO Open Data initiative, licensed under ODC-By 1.0.
