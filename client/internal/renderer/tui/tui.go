package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type FeedMsg []map[string]any

type Column struct {
	Header string
	Format func(data map[string]any) string
	Style  func(value string) lipgloss.Style
}

type model struct {
	devices map[string]map[string]any
	columns []Column
	width   int
	height  int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case FeedMsg:
		m.devices = make(map[string]map[string]any)
		for _, dev := range msg {
			if id, ok := dev["device_id"].(string); ok {
				m.devices[id] = dev
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Align(lipgloss.Center)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1)

	if m.width > 0 {
		border = border.Width(m.width - 2)
	}

	headers := make([]string, len(m.columns))
	for i, c := range m.columns {
		headers[i] = c.Header
	}

	var rows [][]string
	if len(m.devices) == 0 {
		empty := make([]string, len(m.columns))
		empty[0] = "waiting for devices…"
		rows = append(rows, empty)
	} else {
		keys := make([]string, 0, len(m.devices))
		for k := range m.devices {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, id := range keys {
			dev := m.devices[id]
			row := make([]string, len(m.columns))
			for i, col := range m.columns {
				row[i] = col.Format(dev)
			}
			rows = append(rows, row)
		}
	}

	cols := m.columns
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Padding(0, 1)
			}
			if col < len(cols) && row < len(rows) {
				return cols[col].Style(rows[row][col])
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})

	subtitle := dimStyle.Render("q to quit")
	content := headerStyle.Render("statusphere") + "\n\n" + t.Render() + "\n\n" + subtitle
	return border.Render(content)
}

type TUI struct {
	prog *tea.Program
}

func New() *TUI {
	columns := []Column{
		// General

		ColStatus(),
		ColDevice(),

		// -----------------

		// Linux

		// ColCPU(),
		// ColMemory(),
		ColUptime(),
		ColSpotify(),
		// ColLoad(),
		// -----------------

		// Arch
		// ColPackages(),
		// -----------------

		// Hyprland

		// ColWorkspace(),
		ColApp(),
		// -----------------
	}

	m := model{
		devices: make(map[string]map[string]any),
		columns: columns,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	return &TUI{prog: p}
}

func (t *TUI) Run() error {
	_, err := t.prog.Run()
	return err
}

func (t *TUI) UpdateDevices(devices []map[string]any) {
	t.prog.Send(FeedMsg(devices))
}
