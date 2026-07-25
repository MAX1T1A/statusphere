package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type FeedMsg []presence.Snapshot

type tickMsg struct{ id int }

func musicTick(id int) tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{id: id} })
}

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

	cardBorderFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(cFocus).
				Padding(0, 2)

	inputStyle = lipgloss.NewStyle().Foreground(cValue)
	inputCaret = lipgloss.NewStyle().Foreground(cAccent)

	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	notifStyle  = lipgloss.NewStyle().Bold(true).Foreground(cNotify)

	modalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cAccent).
			Padding(1, 4)
	modalTitle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
)

type inputMode int

const (
	modeNone inputMode = iota
	modeMenu
	modeChat
	modeRename
	modeMusic
	modeScreen
)

type menuItem struct {
	label string
	desc  string
}

var menuItems = []menuItem{
	{"Music", "now playing + weekly"},
	{"Screen time", "app usage today"},
	{"Chat", "room messages"},
	{"Rename device", ""},
	{"Quit", ""},
}

const (
	maxNudgeLen  = 128
	maxRenameLen = 32
)

type deviceGroup struct {
	key     string
	devices []presence.Snapshot
}

type model struct {
	groups    []deviceGroup
	blocks    []Block
	custom    Block
	nudges    *NudgeHistory
	spotify   *stats.Cache
	summary   *stats.Cache
	selected  int
	menuIndex int
	tickID    int
	width     int
	height    int

	mode    inputMode
	input   string
	ctrl    Controller
	localID string
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
		if m.mode == modeMenu {
			return m.updateMenu(msg)
		}
		if m.mode == modeMusic || m.mode == modeScreen {
			return m.updateDetail(msg)
		}
		if m.mode != modeNone {
			return m.updateInput(msg), nil
		}

		switch msg.String() {
		case "q", "й", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.groups)-1 {
				m.selected++
			}
		case "enter", " ":
			if len(m.groups) > 0 {
				m.mode = modeMenu
				m.menuIndex = 0
			}
		case "1":
			if len(m.groups) > 0 {
				m.mode = modeMusic
				m.tickID++
				return m, musicTick(m.tickID)
			}
		case "2":
			if len(m.groups) > 0 {
				m.mode = modeScreen
			}
		case "n", "т", "c", "с":
			m.mode = modeChat
			m.input = ""
			m.nudges.MarkRead()
		case "d", "в":
			m.mode = modeRename
			m.input = ""
		}
	case tickMsg:
		if m.mode == modeMusic && msg.id == m.tickID {
			return m, musicTick(msg.id)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case FeedMsg:
		m.groups = groupDevices(msg)
		if m.selected >= len(m.groups) {
			m.selected = len(m.groups) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
	}
	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "й":
		m.mode = modeNone
	case "up", "k":
		if m.menuIndex > 0 {
			m.menuIndex--
		}
	case "down", "j":
		if m.menuIndex < len(menuItems)-1 {
			m.menuIndex++
		}
	case "enter", " ":
		return m.runMenu()
	}
	return m, nil
}

func (m model) runMenu() (tea.Model, tea.Cmd) {
	switch m.menuIndex {
	case 0:
		m.mode = modeMusic
		m.tickID++
		return m, musicTick(m.tickID)
	case 1:
		m.mode = modeScreen
	case 2:
		m.mode = modeChat
		m.input = ""
		m.nudges.MarkRead()
	case 3:
		m.mode = modeRename
		m.input = ""
	case 4:
		return m, tea.Quit
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
	case "backspace":
		if m.mode == modeChat || m.mode == modeRename {
			if runes := []rune(m.input); len(runes) > 0 {
				m.input = string(runes[:len(runes)-1])
			}
		}
	default:
		if m.mode == modeChat || m.mode == modeRename {
			m.typeInput(msg.String())
		}
	}
	return m
}

func (m model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "й":
		m.mode = modeNone
	case "left", "backspace":
		m.mode = modeMenu
	case "s", "ы":
		if m.mode == modeMusic {
			m.syncFocused()
		}
	}
	return m, nil
}

func (m *model) syncFocused() {
	if m.ctrl == nil {
		return
	}
	if uri := m.focusedDevice().String(presence.KeySpotifyURI); uri != "" {
		m.ctrl.SyncSpotify(uri)
	}
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

const minContentWidth = 40

const labelWidth = 7

func clampBox(desired, min, w int) int {
	desired = max(desired, min)
	if limit := w - 4; desired > limit {
		desired = limit
	}
	return max(desired, 10)
}

const modalPad = 8

func modalBoxW(w int) int {
	if w == 0 {
		w = 80
	}
	return clampBox(w*3/5, minContentWidth, w)
}

func scrollWindow(heights []int, selected, avail int) (int, int) {
	n := len(heights)
	if n == 0 {
		return 0, -1
	}
	selected = max(0, min(selected, n-1))

	lo, hi := selected, selected
	used := heights[selected]
	for {
		grew := false
		if hi+1 < n && used+heights[hi+1] <= avail {
			used += heights[hi+1]
			hi++
			grew = true
		}
		if lo-1 >= 0 && used+heights[lo-1] <= avail {
			used += heights[lo-1]
			lo--
			grew = true
		}
		if !grew {
			break
		}
	}
	return lo, hi
}

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

func renderCard(g deviceGroup, blocks []Block, custom Block, focused bool, cardWidth int) string {
	border := cardBorder
	if focused {
		border = cardBorderFocused
	}
	cardPad := border.GetHorizontalBorderSize() + border.GetHorizontalPadding()
	cw := max(cardWidth-cardPad, 12)

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

	textW := max(cw-border.GetHorizontalPadding(), 4)
	lines := strings.Split(strings.Join(sections, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = ansi.Truncate(ln, textW, "…")
	}
	return border.Width(cw).Render(strings.Join(lines, "\n"))
}

func title() string {
	return accentStyle.Render("s") + titleStyle.Render("tatu") + accentStyle.Render("s") + titleStyle.Render("phere")
}

func (m model) View() string {
	switch m.mode {
	case modeMenu:
		return m.menuModal()
	case modeRename:
		return m.renameModal()
	case modeChat:
		return m.chatModal()
	case modeMusic:
		return m.detailModal("music", spotifyDetail(m.focusedDevice(), m.spotify, m.coListeners(), modalBoxW(m.width)-modalPad))
	case modeScreen:
		return m.detailModal("screen time", appDetail(m.focusedDevice(), m.summary))
	}

	width := m.width
	if width == 0 {
		width = 80
	}
	if width < 24 {
		width = 24
	}

	footer := ansi.Truncate(m.footer(), width, "…")

	if len(m.groups) == 0 {
		return title() + "\n\n" + dimStyle.Render("waiting for devices…") + "\n\n" + footer
	}

	cards := make([]string, len(m.groups))
	heights := make([]int, len(m.groups))
	for i, g := range m.groups {
		cards[i] = renderCard(g, m.blocks, m.custom, i == m.selected, width)
		heights[i] = strings.Count(cards[i], "\n") + 1
	}

	avail := max(m.height-6, 3)
	lo, hi := scrollWindow(heights, m.selected, avail)

	var parts []string
	if lo > 0 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("  ↑ %d more", lo)))
	}
	parts = append(parts, lipgloss.JoinVertical(lipgloss.Left, cards[lo:hi+1]...))
	if hi < len(cards)-1 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("  ↓ %d more", len(cards)-1-hi)))
	}

	return title() + "\n\n" + strings.Join(parts, "\n") + "\n\n" + footer
}

func (m model) focusedDevice() presence.Snapshot {
	if m.selected < 0 || m.selected >= len(m.groups) {
		return presence.Snapshot{}
	}
	return m.groups[m.selected].devices[0]
}

func (m model) coListeners() []string {
	d := m.focusedDevice()
	if d.String(presence.KeySpotifyStatus) != "playing" {
		return nil
	}
	key := d.String(presence.KeySpotifyURI)
	if key == "" {
		key = d.String(presence.KeySpotifyTrack)
	}
	if key == "" {
		return nil
	}

	self := d.String(presence.KeyAccountID)
	seen := map[string]bool{}
	var names []string
	for _, g := range m.groups {
		for _, dev := range g.devices {
			if dev.String(presence.KeySpotifyStatus) != "playing" {
				continue
			}
			if acc := dev.String(presence.KeyAccountID); acc != "" && acc == self {
				continue
			}
			k := dev.String(presence.KeySpotifyURI)
			if k == "" {
				k = dev.String(presence.KeySpotifyTrack)
			}
			if k != key {
				continue
			}
			name := dev.String(presence.KeyAccountName)
			if name == "" {
				name = dev.DeviceName()
			}
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

func (m model) focusedName() string {
	d := m.focusedDevice()
	if n := d.String(presence.KeyAccountName); n != "" {
		return n
	}
	if n := d.DeviceName(); n != "" {
		return n
	}
	return "device"
}

func (m model) menuModal() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	boxW := clampBox(w/3, 46, w)

	var rows []string
	for i, it := range menuItems {
		cursor := "  "
		label := dimStyle.Render(it.label)
		if i == m.menuIndex {
			cursor = accentStyle.Render("▸ ")
			label = accentStyle.Render(it.label)
		}
		row := cursor + label
		if it.desc != "" {
			row += dimStyle.Render("   " + it.desc)
		}
		rows = append(rows, ansi.Truncate(row, max(boxW-modalPad, 4), "…"))
	}

	body := modalTitle.Render("menu") + dimStyle.Render(" · "+m.focusedName()) + "\n\n" +
		strings.Join(rows, "\n") + "\n\n" +
		dimStyle.Render("↑↓ select · enter · esc")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalBox.Width(boxW).Render(body))
}

func (m model) detailModal(kind, body string) string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}
	boxW := modalBoxW(w)
	contentW := max(boxW-modalPad, 4)
	if strings.TrimSpace(body) == "" {
		body = dimStyle.Render("nothing here yet")
	}
	lines := strings.Split(body, "\n")
	maxLines := max(h-8, 3)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for i, ln := range lines {
		lines[i] = ansi.Truncate(ln, contentW, "…")
	}
	body = strings.Join(lines, "\n")
	nav := accentStyle.Render("←") + dimStyle.Render(" back · ")
	if kind == "music" {
		nav += accentStyle.Render("s") + dimStyle.Render(" sync · ")
	}
	nav += accentStyle.Render("esc") + dimStyle.Render(" room")

	content := modalTitle.Render(m.focusedName()) + dimStyle.Render(" · "+kind) + "\n\n" +
		body + "\n\n" +
		nav
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalBox.Width(boxW).Render(content))
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

	boxW := modalBoxW(w)
	contentW := max(boxW-modalPad, 4)

	logLines := strings.Split(m.nudges.Render(), "\n")
	maxLines := h - 10
	if maxLines < 3 {
		maxLines = 3
	}
	if len(logLines) > maxLines {
		logLines = logLines[len(logLines)-maxLines:]
	}
	for i, ln := range logLines {
		logLines[i] = ansi.Truncate(ln, contentW, "…")
	}

	body := modalTitle.Render("group chat") + dimStyle.Render("  · everyone in the room") + "\n\n" +
		strings.Join(logLines, "\n") + "\n\n" +
		inputStyle.Render("› "+m.input) + inputCaret.Render("▏") + "\n\n" +
		dimStyle.Render("enter to send · esc to close")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modalBox.Width(boxW).Render(body))
}

func (m model) footer() string {
	hint := dimStyle.Render("↑↓ select · ") + accentStyle.Render("enter") + dimStyle.Render(" menu · ")
	if u := m.nudges.Unread(); u > 0 {
		hint += notifStyle.Render(fmt.Sprintf("● c chat (%d new)", u)) + dimStyle.Render(" · ")
	} else if n := m.nudges.Count(); n > 0 {
		hint += accentStyle.Render("c") + dimStyle.Render(fmt.Sprintf(" chat (%d) · ", n))
	}
	return hint + accentStyle.Render("q") + dimStyle.Render(" quit")
}

type TUI struct {
	prog   *tea.Program
	Nudges *NudgeHistory
}

func newModel(opts Options) model {
	return model{
		blocks:  []Block{BlockSpotify(opts.SpotifyCache), BlockApp(opts.SummaryCache)},
		custom:  BlockCustom(),
		nudges:  NewNudgeHistory(opts.LocalAccountID),
		spotify: opts.SpotifyCache,
		summary: opts.SummaryCache,
		ctrl:    opts.Controller,
		localID: opts.LocalID,
	}
}

func New(opts Options) *TUI {
	m := newModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	return &TUI{prog: p, Nudges: m.nudges}
}

func Snapshot(opts Options, devices []presence.Snapshot, selectDevice, mode string, width, height int) string {
	m := newModel(opts)
	m.groups = groupDevices(devices)
	m.width = width
	m.height = height

	for i, g := range m.groups {
		for _, d := range g.devices {
			if d.DeviceID() == selectDevice {
				m.selected = i
			}
		}
	}

	target := m.focusedDevice().DeviceID()
	if opts.SpotifyCache != nil {
		opts.SpotifyCache.Prime(target)
	}
	if opts.SummaryCache != nil {
		opts.SummaryCache.Prime(target)
	}

	switch mode {
	case "music":
		m.mode = modeMusic
	case "screen":
		m.mode = modeScreen
	}
	return m.View()
}

func (t *TUI) Run() error {
	_, err := t.prog.Run()
	return err
}

func (t *TUI) UpdateDevices(devices []presence.Snapshot) {
	t.prog.Send(FeedMsg(devices))
}
