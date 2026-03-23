package tui

import (
	"fmt"
	"sort"
	"strings"

	"statusphere-client/internal/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FeedMsg []map[string]any

type Block struct {
	Key    string
	Render func(d map[string]any) string
}

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

	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
)

type inputMode int

const (
	modeNone inputMode = iota
	modeNudge
	modeRename
	modeSync
)

const (
	maxNudgeLen  = 128
	maxRenameLen = 32
)

type syncCandidate struct {
	name string
	uri  string
}

type model struct {
	devices map[string]map[string]any
	blocks  []Block
	width   int
	height  int

	mode     inputMode
	input    string
	onNudge  func(string)
	onRename func(string)
	onSync   func(uri string)
	localID  string
	syncList []syncCandidate
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
				m.syncList = nil
			case "backspace":
				if m.mode != modeSync && len(m.input) > 0 {
					runes := []rune(m.input)
					m.input = string(runes[:len(runes)-1])
				}
			default:
				if m.mode == modeSync {
					r := []rune(msg.String())
					if len(r) == 1 && r[0] >= '1' && r[0] <= '9' {
						idx := int(r[0] - '1')
						if idx < len(m.syncList) && m.onSync != nil {
							m.onSync(m.syncList[idx].uri)
						}
						m.mode = modeNone
						m.syncList = nil
					}
				} else {
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
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "n", "т":
			m.mode = modeNudge
			m.input = ""
		case "d", "в":
			m.mode = modeRename
			m.input = ""
		case "s", "ы":
			if m.onSync == nil {
				break
			}
			var candidates []syncCandidate
			for id, dev := range m.devices {
				if id == m.localID {
					continue
				}
				uri, _ := dev["spotify_uri"].(string)
				if uri == "" {
					continue
				}
				name := id
				if n, ok := dev["device_name"].(string); ok && n != "" {
					name = n
				}
				display, _ := dev["spotify_display"].(string)
				if display != "" {
					name += " · " + display
				}
				candidates = append(candidates, syncCandidate{name: name, uri: uri})
			}
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].name < candidates[j].name
			})
			if len(candidates) == 1 {
				m.onSync(candidates[0].uri)
			} else if len(candidates) > 1 {
				m.mode = modeSync
				m.syncList = candidates
			}
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
		h := lipgloss.Height(left)
		nudgeLines := strings.Split(nudgeOut, "\n")
		maxLines := h - 2
		if maxLines < 1 {
			maxLines = 1
		}
		if len(nudgeLines) > maxLines {
			nudgeLines = nudgeLines[len(nudgeLines)-maxLines:]
		}
		right := innerBlock.Height(maxLines).Render(strings.Join(nudgeLines, "\n"))
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

	header := accentStyle.Render("s") + titleStyle.Render("tatu") + accentStyle.Render("s") + titleStyle.Render("phere")
	if len(m.devices) == 0 {
		return outer.Render(
			header + "\n\n" +
				dimStyle.Render("waiting for devices…") + "\n\n" +
				accentStyle.Render("q") + dimStyle.Render("uit"),
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
	case modeSync:
		var opts []string
		for i, c := range m.syncList {
			opts = append(opts, inputStyle.Render(fmt.Sprintf("%d)", i+1))+" "+dimStyle.Render(c.name))
		}
		footer = inputStyle.Render("sync: ") + strings.Join(opts, "  ") + dimStyle.Render("  esc to cancel")
	default:
		footer = accentStyle.Render("n") + dimStyle.Render("udge · ") +
			accentStyle.Render("d") + dimStyle.Render("evice · ") +
			accentStyle.Render("s") + dimStyle.Render("ync · ") +
			accentStyle.Render("q") + dimStyle.Render("uit")
	}

	return outer.Render(
		header + "\n\n" +
			grid + "\n\n" +
			footer,
	)
}

type TUI struct {
	prog   *tea.Program
	Nudges *NudgeHistory
}

func New(spotifyCache, summaryCache *stats.Cache, onNudge, onRename func(string), onSync func(string), localID string) *TUI {
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
		onSync:   onSync,
		localID:  localID,
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
