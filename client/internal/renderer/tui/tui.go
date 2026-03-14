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

	inputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	inputCaret = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

type inputMode int

const (
	modeNone inputMode = iota
	modeNudge
	modeRename
)

const (
	maxNudgeLen  = 60
	maxRenameLen = 32
)

type model struct {
	devices map[string]map[string]any
	blocks  []Block
	width   int
	height  int

	mode     inputMode
	input    string
	onNudge  func(string)
	onRename func(string)
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode != modeNone {
			switch msg.String() {
			case "enter":
				text := strings.TrimSpace(m.input)
				if text != "" {
					switch m.mode {
					case modeNudge:
						if m.onNudge != nil {
							m.onNudge(text)
						}
					case modeRename:
						if m.onRename != nil {
							m.onRename(text)
						}
					}
				}
				m.mode = modeNone
				m.input = ""
			case "esc":
				m.mode = modeNone
				m.input = ""
			case "backspace":
				if len(m.input) > 0 {
					runes := []rune(m.input)
					m.input = string(runes[:len(runes)-1])
				}
			default:
				r := []rune(msg.String())
				if len(r) == 1 {
					limit := maxNudgeLen
					if m.mode == modeRename {
						limit = maxRenameLen
					}
					if len([]rune(m.input)) < limit {
						m.input += msg.String()
					}
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "n":
			m.mode = modeNudge
			m.input = ""
		case "d":
			m.mode = modeRename
			m.input = ""
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
	rendered := make(map[string]string)
	for _, b := range blocks {
		out := b.Render(d)
		if out != "" {
			rendered[b.Key] = out
		}
	}

	var sections []string

	if h, ok := rendered["header"]; ok {
		sections = append(sections, h)
	}

	spotifyOut := rendered["spotify"]
	nudgeOut := rendered["nudge"]

	if spotifyOut != "" && nudgeOut != "" {
		left := innerBlock.Render(spotifyOut)
		right := innerBlock.Render(nudgeOut)
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right))
	} else if spotifyOut != "" {
		sections = append(sections, innerBlock.Render(spotifyOut))
	} else if nudgeOut != "" {
		sections = append(sections, innerBlock.Render(nudgeOut))
	}

	if a, ok := rendered["app"]; ok {
		sections = append(sections, innerBlock.Render(a))
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

	var footer string
	switch m.mode {
	case modeNudge:
		footer = inputStyle.Render("nudge: ") + m.input + inputCaret.Render("█")
	case modeRename:
		footer = inputStyle.Render("name: ") + m.input + inputCaret.Render("█")
	default:
		footer = dimStyle.Render("n nudge · d rename · q quit")
	}

	return outer.Render(
		titleStyle.Render("statusphere") + "\n\n" +
			grid + "\n\n" +
			footer,
	)
}

type TUI struct {
	prog   *tea.Program
	Nudges *NudgeHistory
}

func New(spotifyCache, summaryCache *stats.Cache, onNudge, onRename func(string), localID string) *TUI {
	nudges := NewNudgeHistory(localID)

	blocks := []Block{
		BlockHeader(),
		BlockSpotify(spotifyCache),
		BlockApp(summaryCache),
		BlockNudge(nudges),
	}

	m := model{
		devices:  make(map[string]map[string]any),
		blocks:   blocks,
		onNudge:  onNudge,
		onRename: onRename,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	return &TUI{prog: p, Nudges: nudges}
}

func (t *TUI) Run() error {
	_, err := t.prog.Run()
	return err
}

func (t *TUI) UpdateDevices(devices []map[string]any) {
	t.prog.Send(FeedMsg(devices))
}
