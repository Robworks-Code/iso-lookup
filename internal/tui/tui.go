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

// flatten appends each section depth-first, indenting its title by depth.
func flatten(secs []parse.Section, depth int, out *[]parse.Section) {
	for _, s := range secs {
		row := s
		row.Title = strings.Repeat("  ", depth) + s.Title
		row.Children = nil
		*out = append(*out, row)
		flatten(s.Children, depth+1, out)
	}
}

// New builds the TUI model for a record + parsed document.
func New(rec catalog.Record, doc parse.Document) model {
	var flatSecs []parse.Section
	flatten(doc.Sections, 0, &flatSecs)
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
