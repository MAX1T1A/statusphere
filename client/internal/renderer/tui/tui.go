package tui

import (
	"fmt"
	"sort"
	"strings"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type FeedMsg []presence.Snapshot

type Controller interface {
	Nudge(message string)
	Rename(name string)
	SyncSpotify(uri string)
	SyncCustom(fields []string)
}

type Options struct {
	SpotifyCache   *stats.Cache
	SummaryCache   *stats.Cache
	LocalID        string
	LocalAccountID string
	Controller     Controller
}

type Block struct {
	Render func(d presence.Snapshot) string
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(cName)
	dimStyle   = lipgloss.NewStyle().Foreground(cDim)

	cardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorder).
			Padding(0, 2)

	inputStyle = lipgloss.NewStyle().Foreground(cValue)
	inputCaret = lipgloss.NewStyle().Foreground(cAccent)

	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)

	modalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cAccent).
			Padding(1, 4)
	modalTitle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
)

type inputMode int

const (
	modeNone inputMode = iota
	modeChat
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
	blocks []Block
	custom Block
	nudges *NudgeHistory
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
		case "n", "т", "c", "с":
			m.mode = modeChat
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
		switch m.mode {
		case modeChat:
			if text != "" {
				m.ctrl.Nudge(text)
			}
			m.input = ""
		case modeRename:
			if text != "" {
				m.ctrl.Rename(text)
			}
			m.mode = modeNone
			m.input = ""
		default:
			m.mode = modeNone
			m.input = ""
		}
	case "esc":
		m.mode = modeNone
		m.input = ""
		m.syncDevices = nil
		m.syncTarget = nil
	case "backspace":
		if m.mode == modeChat || m.mode == modeRename {
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
		case modeChat, modeRename:
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

const labelWidth = 7

var sectionLabelStyle = lipgloss.NewStyle().Foreground(cAccent)

func sectionLabel(s string) string {
	return sectionLabelStyle.Render(fmt.Sprintf("%-*s ", labelWidth, s))
}

func durShort(sec int) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func customDivider() string {
	return dimStyle.Render("── custom ──────")
}

func renderCard(g deviceGroup, blocks []Block, custom Block, cardWidth int) string {
	cardPad := cardBorder.GetHorizontalBorderSize() + cardBorder.GetHorizontalPadding()
	cw := cardWidth - cardPad
	if cw < minContentWidth {
		cw = minContentWidth
	}

	d := g.devices[0]
	sections := []string{groupHeader(g)}
	for _, b := range blocks {
		if out := strings.TrimRight(b.Render(d), "\n"); out != "" {
			sections = append(sections, out)
		}
	}
	if custom.Render != nil {
		if out := strings.TrimRight(custom.Render(d), "\n"); out != "" {
			sections = append(sections, customDivider(), out)
		}
	}

	lines := strings.Split(strings.Join(sections, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = ansi.Truncate(ln, cw, "…")
	}
	return cardBorder.Width(cw).Render(strings.Join(lines, "\n"))
}

func title() string {
	return accentStyle.Render("s") + titleStyle.Render("tatu") + accentStyle.Render("s") + titleStyle.Render("phere")
}

func (m model) View() string {
	if m.mode == modeRename {
		return m.renameModal()
	}
	if m.mode == modeChat {
		return m.chatModal()
	}

	width := m.width
	if width < minContentWidth {
		width = minContentWidth
	}

	if len(m.groups) == 0 {
		return title() + "\n\n" + dimStyle.Render("waiting for devices…") + "\n\n" + m.footer()
	}

	var cards []string
	for _, g := range m.groups {
		cards = append(cards, renderCard(g, m.blocks, m.custom, width))
	}
	grid := lipgloss.JoinVertical(lipgloss.Left, cards...)

	return title() + "\n\n" + grid + "\n\n" + m.footer()
}

func (m model) renameModal() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}
	body := modalTitle.Render("rename this device") + "\n\n" +
		inputStyle.Render("› "+m.input) + inputCaret.Render("▏") + "\n\n" +
		dimStyle.Render("enter to save · esc to cancel")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalBox.Render(body))
}

func (m model) chatModal() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	boxW := w * 3 / 5
	if boxW < minContentWidth {
		boxW = minContentWidth
	}

	logLines := strings.Split(m.nudges.Render(), "\n")
	maxLines := h - 10
	if maxLines < 3 {
		maxLines = 3
	}
	if len(logLines) > maxLines {
		logLines = logLines[len(logLines)-maxLines:]
	}
	for i, ln := range logLines {
		logLines[i] = ansi.Truncate(ln, boxW, "…")
	}

	body := modalTitle.Render("group chat") + dimStyle.Render("  · everyone in the room") + "\n\n" +
		strings.Join(logLines, "\n") + "\n\n" +
		inputStyle.Render("› "+m.input) + inputCaret.Render("▏") + "\n\n" +
		dimStyle.Render("enter to send · esc to close")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalBox.Width(boxW).Render(body))
}

func (m model) footer() string {
	switch m.mode {
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
		chat := "hat · "
		if n := m.nudges.Count(); n > 0 {
			chat = fmt.Sprintf("hat (%d) · ", n)
		}
		return accentStyle.Render("c") + dimStyle.Render(chat) +
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
	nudges := NewNudgeHistory(opts.LocalAccountID)

	blocks := []Block{
		BlockSpotify(opts.SpotifyCache),
		BlockApp(opts.SummaryCache),
	}

	m := model{
		blocks:  blocks,
		custom:  BlockCustom(),
		nudges:  nudges,
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
