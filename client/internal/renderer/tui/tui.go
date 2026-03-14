package tui

import (
	"sort"
	"statusphere-client/internal/stats"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FeedMsg []map[string]any

type Block struct {
	Key    string
	Render func(d map[string]any) string
}

const cardWidth = 44

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)

	innerBlock = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)

	outerBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(0, 1)
)

type model struct {
	devices map[string]map[string]any
	blocks  []Block
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

func renderCard(d map[string]any, blocks []Block) string {
	var sections []string
	for _, b := range blocks {
		out := b.Render(d)
		if out == "" {
			continue
		}
		if b.Key == "header" {
			sections = append(sections, out)
		} else {
			sections = append(sections, innerBlock.Render(out))
		}
	}
	return cardBorder.Render(strings.Join(sections, "\n"))
}

func (m model) View() string {
	outer := outerBorder
	if m.width > 0 {
		outer = outer.Width(m.width - 2)
	}

	if len(m.devices) == 0 {
		return outer.Render(
			titleStyle.Render("statusphere") + "\n\n" +
				dimStyle.Render("waiting for devices…") + "\n\n" +
				dimStyle.Render("q to quit"),
		)
	}

	keys := make([]string, 0, len(m.devices))
	for k := range m.devices {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var cards []string
	for _, id := range keys {
		cards = append(cards, renderCard(m.devices[id], m.blocks))
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, cards...)

	return outer.Render(
		titleStyle.Render("statusphere") + "\n\n" +
			grid + "\n\n" +
			dimStyle.Render("q to quit"),
	)
}

type TUI struct {
	prog *tea.Program
}

func New(spotifyCache, summaryCache *stats.Cache) *TUI {
	blocks := []Block{
		BlockHeader(),
		BlockSpotify(spotifyCache),
		BlockApp(summaryCache),
	}

	m := model{
		devices: make(map[string]map[string]any),
		blocks:  blocks,
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
