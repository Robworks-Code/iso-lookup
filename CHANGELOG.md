# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-06-05

### Added
- `scan` recommendations now lead with a confidence marker (`●` high / `◐`
  medium / `○` low) so the most relevant standards stand out at a glance; the
  markers stay distinguishable under `--no-color` because they differ by fill.
- `scan` prints a "Start here" block listing the top published, curated
  standards as ready-to-run `iso show <ref>` commands.
- `show` flags superseded or withdrawn standards with a caveat banner, and
  prints a copy-pasteable `Next →` action line (`iso browse` / `iso open`).

### Changed
- `scan` now groups by category by default (was component), deduplicating
  standards across components and reading as concern domains; `--group-by
  component|ics` remain available.
- `scan` states each rationale once per group instead of repeating it on every
  standard line, recovers the driving components in a dim `from …` header
  annotation, and shortens long ISO titles in the default view (full titles
  retained under `--long`).

## [0.3.0] - 2026-06-04

### Added
- Color-coded, formatted output across `search`, `show`, `scan`, `config`, and
  `update`: references, statuses (green published / red withdrawn / yellow under
  review), confidence, group headers, rationale, and evidence are visually
  distinct. The `scan` header is framed in a panel.
- `scan --interactive` / `-i`: a two-pane TUI to browse the grouped report
  (groups left, standards + rationale right; arrows/`j`/`k` switch groups,
  `f`/`b` scroll, `q` quits).
- Global `--no-color` and `--color` flags. Color is auto-disabled when output is
  piped or redirected and honors the `NO_COLOR` convention.
- `config set-docs` / `set-index` / `set-pager` now print a confirmation line on
  success; `config` shows a dim `(not set)` for unset values.

### Changed
- Listings use width-aware alignment that stays correct with color and never
  wraps over-long values onto extra lines.

## [0.2.0] - 2026-06-03

### Added
- `iso scan [path]`: detect a project's technology stack from marker files and
  dependency manifests, then recommend relevant current ISO standards, grouped by
  component, category, or ICS code. Includes `scan stack` (components only) and
  `scan why <term>` (explain what drove a recommendation), plus `--group-by`,
  `--category`, `--component`, `--discover`, `--include-drafts`, `--limit`,
  `--depth`, `--sort`, `--long`, and `--json` flags.

## [0.1.1] - 2026-06-03

### Changed
- Release pipeline publishes the Homebrew cask using a token minted at runtime
  from a GitHub App, removing the need to rotate a personal access token.

## [0.1.0] - 2026-06-02

### Added
- Initial `iso` CLI: `update` (download/refresh the offline ISO Open Data index),
  `search` (keyword search with filters and `--sort`), `show` (metadata summary +
  table of contents), `chapter` (print a chapter from a local file), `open` (open
  the official ISO page), `browse` (interactive chapter TUI), and `config`.
- Result filtering (`--ics`, `--committee`, `--status`, `--year`, `--published`),
  sorting, `--limit`/`--count`, `--json` output, and richer help text.
- Distribution via Homebrew (GoReleaser-built cask), `go install`, prebuilt
  release binaries (darwin/linux × amd64/arm64), and a `Makefile`.

[Unreleased]: https://github.com/Robworks-Code/iso-lookup/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/Robworks-Code/iso-lookup/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Robworks-Code/iso-lookup/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/Robworks-Code/iso-lookup/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Robworks-Code/iso-lookup/releases/tag/v0.1.0
