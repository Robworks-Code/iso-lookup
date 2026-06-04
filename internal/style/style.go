// Package style holds the shared terminal theme for iso's text output and TUIs.
// It centralizes the lipgloss styles and the status/confidence color logic so
// the render package and the interactive browsers (internal/tui) draw from one
// palette.
//
// Colors use the ANSI-16 palette (codes "8"–"15") rather than hex so they honor
// the user's terminal theme and degrade gracefully on 4-bit terminals. Color is
// applied lazily by lipgloss's default renderer, which detects the terminal and
// respects NO_COLOR/CLICOLOR_FORCE; cmd/iso layers an explicit --no-color/--color
// override on top via lipgloss.SetColorProfile.
package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Column slot widths (content + a ~2-col trailing gap) for listing layouts.
const (
	RefW       = 30
	StatusW    = 13
	DateW      = 12
	CommitteeW = 24
	NameW      = 24
)

// Pad left-aligns styled text (which may carry ANSI escapes) into a column of
// width visible columns, measuring with ANSI stripped so color never throws off
// alignment. It never wraps or truncates: content wider than the slot stays on
// one line with a 2-space separator, matching the CLI's original overflow
// behavior (unlike a fixed-width lipgloss cell, which would wrap).
func Pad(styled string, width int) string {
	w := lipgloss.Width(styled)
	if w >= width {
		return styled + "  "
	}
	return styled + strings.Repeat(" ", width-w)
}

// Named styles for the common output roles.
var (
	// Header is a plain bold heading (e.g. a document title).
	Header = lipgloss.NewStyle().Bold(true)
	// SubHeader marks section/group headers with a bold blue accent.
	SubHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	// Panel frames a short banner (e.g. the scan header line).
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)
	// Ref styles an ISO reference in bold cyan.
	Ref = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	// Dim is muted grey for secondary text (evidence, tags, counts, notices).
	Dim = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	// Rationale styles a "why" explanation in muted italic.
	Rationale = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Italic(true)
	// Summary highlights a closing summary line in bold green.
	Summary = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	// Label is a fixed-width grey field label so values align (in show output).
	Label = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	// URL renders a link in cyan. (No underline: lipgloss underlines rune-by-rune,
	// which bloats the output stream with little visual gain.)
	URL = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	// Warn flags a caveat (truncation, drafts) in yellow.
	Warn = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// Status returns the color style for a catalog status label: green for
// published, red for withdrawn, yellow for under-review, grey for drafts/other.
func Status(status string) lipgloss.Style {
	s := strings.ToLower(status)
	switch {
	case strings.Contains(s, "withdrawn"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	case strings.Contains(s, "review"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	case strings.Contains(s, "published"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	}
}

// Confidence returns the color style for a "low"/"medium"/"high" confidence
// label: green high, yellow medium, grey low.
func Confidence(level string) lipgloss.Style {
	switch strings.ToLower(level) {
	case "high":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	case "medium":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	}
}
