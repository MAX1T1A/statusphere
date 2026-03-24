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
	modeSyncDevice
	modeSyncAction
)

const (
	maxNudgeLen  = 128
	maxRenameLen = 32
)

type syncDevice struct {
	id     string
	name   string
	uri    string
	fields []string
}

type model struct {
	devices map[string]map[string]any
	blocks  []Block
	width   int
	height  int

	mode         inputMode
	input        string
	onNudge      func(string)
	onRename     func(string)
	onSync       func(uri string)
	onSyncCustom func(fields []string)
	localID      string
	syncDevices  []syncDevice
	syncTarget   *syncDevice
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
				m.syncDevices = nil
				m.syncTarget = nil
			case "backspace":
				if m.mode == modeNudge || m.mode == modeRename {
					if len(m.input) > 0 {
						runes := []rune(m.input)
						m.input = string(runes[:len(runes)-1])
					}
				}
			default:
				switch m.mode {
				case modeSyncDevice:
					r := []rune(msg.String())
					if len(r) == 1 && r[0] >= '1' && r[0] <= '9' {
						idx := int(r[0] - '1')
						if idx < len(m.syncDevices) {
							dev := m.syncDevices[idx]
							if dev.uri != "" && len(dev.fields) > 0 {
								m.syncTarget = &dev
								m.mode = modeSyncAction
							} else if dev.uri != "" {
								if m.onSync != nil {
									m.onSync(dev.uri)
								}
								m.mode = modeNone
							} else if len(dev.fields) > 0 {
								if m.onSyncCustom != nil {
									m.onSyncCustom(dev.fields)
								}
								m.mode = modeNone
							}
							m.syncDevices = nil
						}
					}
				case modeSyncAction:
					switch msg.String() {
					case "1":
						if m.syncTarget != nil && m.syncTarget.uri != "" && m.onSync != nil {
							m.onSync(m.syncTarget.uri)
						}
						m.mode = modeNone
						m.syncTarget = nil
					case "2":
						if m.syncTarget != nil && len(m.syncTarget.fields) > 0 && m.onSyncCustom != nil {
							m.onSyncCustom(m.syncTarget.fields)
						}
						m.mode = modeNone
						m.syncTarget = nil
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
			devices := m.buildSyncDevices()
			if len(devices) == 0 {
				break
			}
			if len(devices) == 1 {
				dev := devices[0]
				if dev.uri != "" && len(dev.fields) > 0 {
					m.syncTarget = &dev
					m.mode = modeSyncAction
				} else if dev.uri != "" {
					if m.onSync != nil {
						m.onSync(dev.uri)
					}
				} else if len(dev.fields) > 0 {
					if m.onSyncCustom != nil {
						m.onSyncCustom(dev.fields)
					}
				}
			} else {
				m.syncDevices = devices
				m.mode = modeSyncDevice
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

func (m model) buildSyncDevices() []syncDevice {
	var devices []syncDevice
	for id, dev := range m.devices {
		if id == m.localID {
			continue
		}
		uri, _ := dev["spotify_uri"].(string)
		var fields []string
		if raw, ok := dev["custom_fields"].([]any); ok {
			for _, v := range raw {
				if s, ok := v.(string); ok && s != "" {
					fields = append(fields, s)
				}
			}
		}
		if uri == "" && len(fields) == 0 {
			continue
		}
		name := id
		if n, ok := dev["device_name"].(string); ok && n != "" {
			name = n
		}
		devices = append(devices, syncDevice{id: id, name: name, uri: uri, fields: fields})
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].name < devices[j].name
	})
	return devices
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

	if c, ok := rendered["custom"]; ok {
		sections = append(sections, innerBlock.Render(c))
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
	case modeSyncDevice:
		var opts []string
		for i, d := range m.syncDevices {
			opts = append(opts, inputStyle.Render(fmt.Sprintf("%d)", i+1))+" "+dimStyle.Render(d.name))
		}
		footer = inputStyle.Render("sync from: ") + strings.Join(opts, "  ") + dimStyle.Render("  esc to cancel")
	case modeSyncAction:
		name := ""
		if m.syncTarget != nil {
			name = m.syncTarget.name
		}
		var opts []string
		if m.syncTarget != nil && m.syncTarget.uri != "" {
			opts = append(opts, accentStyle.Render("1")+dimStyle.Render(") spotify"))
		}
		if m.syncTarget != nil && len(m.syncTarget.fields) > 0 {
			opts = append(opts, accentStyle.Render("2")+dimStyle.Render(") custom fields"))
		}
		footer = inputStyle.Render(name+": ") + strings.Join(opts, "  ") + dimStyle.Render("  esc to cancel")
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

func New(spotifyCache, summaryCache *stats.Cache, onNudge, onRename func(string), onSync func(string), onSyncCustom func([]string), localID string) *TUI {
	nudges := NewNudgeHistory(localID)

	blocks := []Block{
		BlockHeader(),
		BlockSpotify(spotifyCache),
		BlockApp(summaryCache),
		BlockCustom(),
		BlockNudge(nudges),
	}

	m := model{
		devices:      make(map[string]map[string]any),
		blocks:       blocks,
		onNudge:      onNudge,
		onRename:     onRename,
		onSync:       onSync,
		onSyncCustom: onSyncCustom,
		localID:      localID,
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
