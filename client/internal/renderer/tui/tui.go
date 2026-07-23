package tui

import (
	"fmt"
	"sort"
	"strings"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FeedMsg []presence.Snapshot

type Controller interface {
	Nudge(message string)
	Rename(name string)
	SyncSpotify(uri string)
	SyncCustom(fields []string)
}

type Options struct {
	SpotifyCache *stats.Cache
	SummaryCache *stats.Cache
	LocalID      string
	CustomOrder  []string
	Controller   Controller
}

type Block struct {
	Render func(d presence.Snapshot) string
}

type LayoutRow struct {
	Blocks []Block
	Bare   bool
	Anchor int
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

type deviceGroup struct {
	key     string
	devices []presence.Snapshot
}

type model struct {
	groups []deviceGroup
	layout []LayoutRow
	width  int
	height int

	mode        inputMode
	input       string
	ctrl        Controller
	localID     string
	syncDevices []syncDevice
	syncTarget  *syncDevice
}

func groupDevices(snaps []presence.Snapshot) []deviceGroup {
	byKey := make(map[string][]presence.Snapshot)
	var order []string
	for _, d := range snaps {
		key := d.String(presence.KeyAccountID)
		if key == "" {
			key = d.DeviceID()
		}
		if key == "" {
			continue
		}
		if _, ok := byKey[key]; !ok {
			order = append(order, key)
		}
		byKey[key] = append(byKey[key], d)
	}

	groups := make([]deviceGroup, 0, len(order))
	for _, key := range order {
		devs := byKey[key]
		sort.Slice(devs, func(i, j int) bool {
			li, _ := devs[i].Int(presence.KeyLastSeen)
			lj, _ := devs[j].Int(presence.KeyLastSeen)
			return li > lj
		})
		groups = append(groups, deviceGroup{key: key, devices: devs})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].key < groups[j].key })
	return groups
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode != modeNone {
			return m.updateInput(msg), nil
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
			m.startSync()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case FeedMsg:
		m.groups = groupDevices(msg)
	}
	return m, nil
}

func (m model) updateInput(msg tea.KeyMsg) model {
	switch msg.String() {
	case "enter":
		text := strings.TrimSpace(m.input)
		if text != "" {
			switch m.mode {
			case modeNudge:
				m.ctrl.Nudge(text)
			case modeRename:
				m.ctrl.Rename(text)
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
			if runes := []rune(m.input); len(runes) > 0 {
				m.input = string(runes[:len(runes)-1])
			}
		}
	default:
		switch m.mode {
		case modeSyncDevice:
			m.pickSyncDevice(msg.String())
		case modeSyncAction:
			m.pickSyncAction(msg.String())
		default:
			m.typeInput(msg.String())
		}
	}
	return m
}

func (m *model) typeInput(s string) {
	if r := []rune(s); len(r) != 1 {
		return
	}
	limit := maxNudgeLen
	if m.mode == modeRename {
		limit = maxRenameLen
	}
	if len([]rune(m.input)) < limit {
		m.input += s
	}
}

func (m *model) startSync() {
	devices := m.buildSyncDevices()
	switch len(devices) {
	case 0:
		return
	case 1:
		m.applySync(devices[0])
	default:
		m.syncDevices = devices
		m.mode = modeSyncDevice
	}
}

func (m *model) applySync(dev syncDevice) {
	switch {
	case dev.uri != "" && len(dev.fields) > 0:
		m.syncTarget = &dev
		m.mode = modeSyncAction
	case dev.uri != "":
		m.ctrl.SyncSpotify(dev.uri)
	case len(dev.fields) > 0:
		m.ctrl.SyncCustom(dev.fields)
	}
}

func (m *model) pickSyncDevice(key string) {
	r := []rune(key)
	if len(r) != 1 || r[0] < '1' || r[0] > '9' {
		return
	}
	idx := int(r[0] - '1')
	if idx >= len(m.syncDevices) {
		return
	}
	dev := m.syncDevices[idx]
	m.syncDevices = nil
	if dev.uri != "" && len(dev.fields) > 0 {
		m.syncTarget = &dev
		m.mode = modeSyncAction
		return
	}
	m.applySync(dev)
	if m.mode == modeSyncDevice {
		m.mode = modeNone
	}
}

func (m *model) pickSyncAction(key string) {
	if m.syncTarget == nil {
		return
	}
	switch key {
	case "1":
		if m.syncTarget.uri != "" {
			m.ctrl.SyncSpotify(m.syncTarget.uri)
		}
	case "2":
		if len(m.syncTarget.fields) > 0 {
			m.ctrl.SyncCustom(m.syncTarget.fields)
		}
	default:
		return
	}
	m.mode = modeNone
	m.syncTarget = nil
}

func (m model) buildSyncDevices() []syncDevice {
	var devices []syncDevice
	for _, g := range m.groups {
		for _, dev := range g.devices {
			id := dev.DeviceID()
			if id == "" || id == m.localID {
				continue
			}
			uri := dev.String(presence.KeySpotifyURI)
			fields := dev.Strings(presence.KeyCustomFields)
			if uri == "" && len(fields) == 0 {
				continue
			}
			name := id
			if n := dev.DeviceName(); n != "" {
				name = n
			}
			devices = append(devices, syncDevice{id: id, name: name, uri: uri, fields: fields})
		}
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].name < devices[j].name
	})
	return devices
}

const minContentWidth = 40

func renderCard(g deviceGroup, layout []LayoutRow, cardWidth int) string {
	cardPad := cardBorder.GetHorizontalBorderSize() + cardBorder.GetHorizontalPadding()
	innerPad := innerBlock.GetHorizontalBorderSize() + innerBlock.GetHorizontalPadding()

	cw := cardWidth - cardPad
	if cw < minContentWidth {
		cw = minContentWidth
	}

	d := g.devices[0]
	sections := []string{groupHeader(g)}

	for _, row := range layout {
		type activeBlock struct {
			content string
			origIdx int
		}
		var active []activeBlock
		for i, b := range row.Blocks {
			out := b.Render(d)
			if out != "" {
				active = append(active, activeBlock{content: out, origIdx: i})
			}
		}
		if len(active) == 0 {
			continue
		}

		if row.Bare {
			var texts []string
			for _, a := range active {
				texts = append(texts, a.content)
			}
			sections = append(sections, strings.Join(texts, "\n"))
			continue
		}

		if len(active) == 1 {
			iw := cw - innerPad
			if iw < 1 {
				iw = 1
			}
			sections = append(sections, innerBlock.Width(iw).Render(active[0].content))
			continue
		}

		gap := 1
		n := len(active)
		totalOuter := cw - gap*(n-1)
		base := totalOuter / n

		innerVPad := innerBlock.GetVerticalBorderSize() + innerBlock.GetVerticalPadding()

		iws := make([]int, n)
		usedWidth := 0
		rendered := make([]string, n)
		for i, a := range active {
			remaining := cw - usedWidth - gap*(n-1-i)
			targetOuter := base
			if i == n-1 {
				targetOuter = remaining
			} else if targetOuter > remaining {
				targetOuter = remaining
			}
			iw := targetOuter - innerPad
			if iw < 1 {
				iw = 1
			}
			iws[i] = iw
			r := innerBlock.Width(iw).Render(a.content)
			rendered[i] = r
			usedWidth += lipgloss.Width(r) + gap
		}

		anchorH := 0
		anchorFound := false
		for i, a := range active {
			if a.origIdx == row.Anchor {
				anchorH = lipgloss.Height(rendered[i])
				anchorFound = true
				break
			}
		}
		if !anchorFound {
			for _, r := range rendered {
				if h := lipgloss.Height(r); h > anchorH {
					anchorH = h
				}
			}
		}

		ih := anchorH - innerVPad
		if ih < 1 {
			ih = 1
		}

		var parts []string
		usedWidth = 0
		for i, a := range active {
			iw := iws[i]
			if i == n-1 {
				remaining := cw - usedWidth - gap*(n-1-i)
				iw = remaining - innerPad
				if iw < 1 {
					iw = 1
				}
			}
			content := a.content
			if anchorFound && a.origIdx != row.Anchor {
				lines := strings.Split(content, "\n")
				if len(lines) > ih {
					lines = lines[len(lines)-ih:]
					content = strings.Join(lines, "\n")
				}
			}
			r := innerBlock.Width(iw).Height(ih).Render(content)
			parts = append(parts, r)
			usedWidth += lipgloss.Width(r) + gap
		}

		var joined []string
		for i, p := range parts {
			if i > 0 {
				joined = append(joined, " ")
			}
			joined = append(joined, p)
		}
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, joined...))
	}

	return cardBorder.Width(cw).Render(strings.Join(sections, "\n"))
}

func (m model) View() string {
	outerPad := outerBorder.GetHorizontalBorderSize() + outerBorder.GetHorizontalPadding()

	totalW := m.width
	if totalW < minContentWidth+outerPad {
		totalW = minContentWidth + outerPad
	}

	outer := outerBorder.Width(totalW - outerPad)

	contentW := totalW - outerPad
	cardWidth := contentW

	header := accentStyle.Render("s") + titleStyle.Render("tatu") + accentStyle.Render("s") + titleStyle.Render("phere")
	if len(m.groups) == 0 {
		return outer.Render(
			header + "\n\n" +
				dimStyle.Render("waiting for devices…") + "\n\n" +
				accentStyle.Render("q") + dimStyle.Render("uit"),
		)
	}

	var cards []string
	for _, g := range m.groups {
		cards = append(cards, renderCard(g, m.layout, cardWidth))
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, cards...)

	return outer.Render(header + "\n\n" + grid + "\n\n" + m.footer())
}

func (m model) footer() string {
	switch m.mode {
	case modeNudge:
		return inputStyle.Render("nudge: ") + m.input + inputCaret.Render("█")
	case modeRename:
		return inputStyle.Render("name: ") + m.input + inputCaret.Render("█")
	case modeSyncDevice:
		var opts []string
		for i, d := range m.syncDevices {
			opts = append(opts, inputStyle.Render(fmt.Sprintf("%d)", i+1))+" "+dimStyle.Render(d.name))
		}
		return inputStyle.Render("sync from: ") + strings.Join(opts, "  ") + dimStyle.Render("  esc to cancel")
	case modeSyncAction:
		name := ""
		var opts []string
		if m.syncTarget != nil {
			name = m.syncTarget.name
			if m.syncTarget.uri != "" {
				opts = append(opts, accentStyle.Render("1")+dimStyle.Render(") spotify"))
			}
			if len(m.syncTarget.fields) > 0 {
				opts = append(opts, accentStyle.Render("2")+dimStyle.Render(") custom fields"))
			}
		}
		return inputStyle.Render(name+": ") + strings.Join(opts, "  ") + dimStyle.Render("  esc to cancel")
	default:
		return accentStyle.Render("n") + dimStyle.Render("udge · ") +
			accentStyle.Render("d") + dimStyle.Render("evice · ") +
			accentStyle.Render("s") + dimStyle.Render("ync · ") +
			accentStyle.Render("q") + dimStyle.Render("uit")
	}
}

type TUI struct {
	prog   *tea.Program
	Nudges *NudgeHistory
}

func New(opts Options) *TUI {
	nudges := NewNudgeHistory(opts.LocalID)

	layout := []LayoutRow{
		{Blocks: []Block{BlockCustom(opts.CustomOrder)}},
		{Blocks: []Block{BlockSpotify(opts.SpotifyCache), BlockNudge(nudges)}, Anchor: 0},
		{Blocks: []Block{BlockApp(opts.SummaryCache)}},
	}

	m := model{
		layout:  layout,
		ctrl:    opts.Controller,
		localID: opts.LocalID,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	return &TUI{prog: p, Nudges: nudges}
}

func (t *TUI) Run() error {
	_, err := t.prog.Run()
	return err
}

func (t *TUI) UpdateDevices(devices []presence.Snapshot) {
	t.prog.Send(FeedMsg(devices))
}
