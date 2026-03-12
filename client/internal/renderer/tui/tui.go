package tui

import (
	"fmt"
	"sort"
	"strings"

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
	mouseX  int
	mouseY  int
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
	case tea.MouseMsg:
		m.mouseX = msg.X
		m.mouseY = msg.Y
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

	keys := make([]string, 0, len(m.devices))
	for k := range m.devices {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var rows [][]string
	if len(m.devices) == 0 {
		empty := make([]string, len(m.columns))
		empty[0] = "waiting for devices…"
		rows = append(rows, empty)
	} else {
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

	// Tooltip: border-top(1) + title(1) + blank(1) + table-top-border(1) + header(1) + separator(1) = 6
	const tableDataStartY = 6
	rowIdx := m.mouseY - tableDataStartY
	tooltipLine := ""
	if rowIdx >= 0 && rowIdx < len(keys) {
		dev := m.devices[keys[rowIdx]]
		if tip := buildTooltip(dev); tip != "" {
			tooltipLine = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("14")).
				Padding(0, 1).
				Render(tip)
		}
	}

	subtitle := dimStyle.Render("q to quit")
	content := headerStyle.Render("statusphere") + "\n\n" + t.Render() + "\n\n" + subtitle + "\n" + tooltipLine
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

		ColCPU(),
		ColMemory(),
		ColMusic(),
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
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	return &TUI{prog: p}
}

func (t *TUI) Run() error {
	_, err := t.prog.Run()
	return err
}

func (t *TUI) UpdateDevices(devices []map[string]any) {
	t.prog.Send(FeedMsg(devices))
}

func buildTooltip(dev map[string]any) string {
	var parts []string

	if name, ok := dev["device_name"].(string); ok && name != "" {
		if len([]rune(name)) > maxDeviceLen {
			parts = append(parts, fmt.Sprintf("device: %s", name))
		}
	}

	app, _ := dev["active_app"].(string)
	win, _ := dev["active_window"].(string)
	if app != "" {
		app = CleanAppName(app)
	}
	if app != "" || win != "" {
		appFull := app
		winFull := win
		if app != "" {
			winFull = cleanTitle(win, app)
		}
		displayApp := appFull
		if winFull != "" {
			displayApp = appFull + " · " + winFull
		}
		truncatedDisplay := ""
		if appFull == "" {
			truncatedDisplay = truncate(winFull, maxAppLen)
		} else if winFull == "" {
			truncatedDisplay = truncate(appFull, maxAppLen)
		} else {
			truncatedDisplay = appFull + " · " + truncate(winFull, maxAppLen-len([]rune(appFull))-3)
		}
		if displayApp != truncatedDisplay {
			parts = append(parts, fmt.Sprintf("app: %s", displayApp))
		}
	}

	return strings.Join(parts, "  │  ")
}
