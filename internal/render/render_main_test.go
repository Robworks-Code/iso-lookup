package render

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/Robworks-Code/iso-lookup/internal/style"
)

// TestMain forces the no-color (Ascii) profile so the plain-text assertions in
// the other tests hold regardless of the environment running them. Width-aware
// alignment is exercised separately under a color profile in
// TestColumnWidthIgnoresColor.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

// TestPadIgnoresColor proves style.Pad aligns by printable width even when a
// cell carries ANSI color — the reason we replaced fmt's %-Ns padding, which
// counts the escape bytes and misaligns. A short colored cell occupies exactly
// the slot width; an over-long cell stays on one line (never wraps) with a
// 2-space separator.
func TestPadIgnoresColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	short := style.Pad(style.Ref.Render("ISO 9001:2015"), style.RefW)
	if w := lipgloss.Width(short); w != style.RefW {
		t.Errorf("padded short cell width = %d; want %d", w, style.RefW)
	}
	if !strings.Contains(short, "\x1b[") {
		t.Error("expected ANSI escapes in the colored cell under the ANSI256 profile")
	}

	// An over-long value must not wrap (stays on one line).
	overlong := style.Pad(style.Status("New project approved").Render("New project approved"), style.StatusW)
	if strings.Contains(overlong, "\n") {
		t.Errorf("over-long cell wrapped: %q", overlong)
	}
}
