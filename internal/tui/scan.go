package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Robworks-Code/iso-lookup/internal/scan"
	"github.com/Robworks-Code/iso-lookup/internal/style"
)

// scanModel is a two-pane browser for a scan report: a selectable list of groups
// on the left, the selected group's recommendations (with status, rationale, and
// evidence) on the right.
type scanModel struct {
	rep    scan.Report
	cursor int // selected group
	scroll int // line offset within the right pane
	width  int
	height int
}

func newScan(rep scan.Report) scanModel { return scanModel{rep: rep} }

func (m scanModel) Init() tea.Cmd { return nil }

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.rep.Groups)-1 {
				m.cursor++
				m.scroll = 0
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.scroll = 0
			}
		case "pgdown", "f", " ":
			m.scroll += m.bodyHeight() / 2
		case "pgup", "b":
			m.scroll -= m.bodyHeight() / 2
			if m.scroll < 0 {
				m.scroll = 0
			}
		}
	}
	return m, nil
}

// bodyHeight is the number of body rows available in the right pane, leaving room
// for the footer.
func (m scanModel) bodyHeight() int {
	h := m.height - 2
	if h < 1 {
		return 1
	}
	return h
}

var (
	scanListStyle = lipgloss.NewStyle().Width(30).
			Border(lipgloss.NormalBorder(), false, true, false, false).Padding(0, 1)
	scanBodyStyle = lipgloss.NewStyle().Padding(0, 2)
)

func (m scanModel) View() string {
	if len(m.rep.Groups) == 0 {
		return "No groups to browse. Press q to quit."
	}

	// Left: one row per group, "Header (N)", selected row accented.
	var left strings.Builder
	for i, g := range m.rep.Groups {
		label := fmt.Sprintf("%s %s", g.Header, style.Dim.Render(fmt.Sprintf("(%d)", len(g.Recommendations))))
		if i == m.cursor {
			left.WriteString(style.SubHeader.Render("> "+g.Header) + style.Dim.Render(fmt.Sprintf(" (%d)", len(g.Recommendations))))
		} else {
			left.WriteString("  " + label)
		}
		left.WriteString("\n")
	}

	bodyWidth := m.width - 34
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	lines := groupLines(m.rep.Groups[m.cursor], bodyWidth)

	// Clamp scroll and take a window of body lines.
	maxScroll := len(lines) - m.bodyHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	end := m.scroll + m.bodyHeight()
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines
	if m.scroll < len(lines) {
		visible = lines[m.scroll:end]
	}

	footer := footerHint(m.scroll < maxScroll || m.scroll > 0)
	right := scanBodyStyle.Render(strings.Join(visible, "\n") + "\n" + footer)
	return lipgloss.JoinHorizontal(lipgloss.Top, scanListStyle.Render(left.String()), right)
}

// groupLines renders one group's recommendations into styled lines, wrapping
// rationale to the available width.
func groupLines(g scan.Group, width int) []string {
	var lines []string
	header := style.SubHeader.Render(g.Header)
	if g.Total > len(g.Recommendations) {
		header += style.Dim.Render(fmt.Sprintf("  (showing %d of %d)", len(g.Recommendations), g.Total))
	}
	lines = append(lines, header, "")

	if len(g.Recommendations) == 0 {
		lines = append(lines, style.Dim.Render("(no standards in the catalog yet)"))
	}
	wrap := lipgloss.NewStyle().Width(width)
	for _, rec := range g.Recommendations {
		r := rec.Record
		title := r.Title
		if rec.Discovered {
			title += style.Dim.Render("  (discovered)")
		}
		head := fmt.Sprintf("%s  %s  %s", style.Ref.Render(r.Reference), style.Status(r.Status).Render(r.Status), title)
		lines = append(lines, wrap.Render(head))
		if rec.Rationale != "" {
			lines = append(lines, splitLines(wrap.Render(style.Rationale.Render(rec.Rationale)))...)
		}
		if len(rec.Components) > 0 {
			lines = append(lines, style.Dim.Render("  driven by: "+strings.Join(rec.Components, ", ")))
		}
		lines = append(lines, "")
	}
	if len(g.Missing) > 0 {
		lines = append(lines, style.Dim.Render("not yet in catalog: "+strings.Join(g.Missing, ", ")))
	}
	return lines
}

func splitLines(s string) []string { return strings.Split(s, "\n") }

func footerHint(scrollable bool) string {
	hint := "[↑/↓ groups · q quit]"
	if scrollable {
		hint = "[↑/↓ groups · f/b scroll · q quit]"
	}
	return "\n" + style.Dim.Render(hint)
}

// RunScan launches the interactive scan-report browser.
func RunScan(rep scan.Report) error {
	_, err := tea.NewProgram(newScan(rep), tea.WithAltScreen()).Run()
	return err
}
