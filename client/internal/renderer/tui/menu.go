package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/privacy"
	"statusphere-client/internal/version"
)

type rowKind int

const (
	rowAction rowKind = iota
	rowSection
	rowCheck
)

type menuRow struct {
	kind    rowKind
	id      string
	label   string
	desc    string
	checked bool
	open    bool
}

func (m model) menuTitle() string {
	switch m.mode {
	case modeSettings:
		return "settings"
	case modeView:
		return "panel"
	}
	return m.focusedName()
}

func (m model) menuRows() []menuRow {
	switch m.mode {
	case modeSettings:
		return m.settingsRows()
	case modeView:
		return m.panelRows()
	}
	return m.personRows()
}

func (m model) personRows() []menuRow {
	account := m.focusedDevice().String(presence.KeyAccountID)
	open := m.openSection == "music"

	rows := []menuRow{{kind: rowSection, id: "music", label: "Music", desc: "pick what to show", open: open}}
	if open {
		for _, p := range musicPieces {
			rows = append(rows, menuRow{
				kind:    rowCheck,
				id:      p.id,
				label:   p.label,
				desc:    p.desc,
				checked: m.expand.enabled(account, p.id),
			})
		}
	}
	rows = append(rows, menuRow{kind: rowAction, id: "screen", label: "Screen time", desc: "app usage today"})
	if m.focusedDevice().String(presence.KeySpotifyURI) != "" {
		rows = append(rows, menuRow{kind: rowAction, id: "sync", label: "Play the same track", desc: "open it in your spotify"})
	}
	if account != "" && account != m.chat.localID {
		rows = append(rows, menuRow{kind: rowAction, id: "message", label: "Message " + m.focusedName(), desc: "direct message"})
	}
	return rows
}

func (m model) settingsRows() []menuRow {
	update := menuRow{kind: rowAction, id: "update", label: "Check for updates", desc: version.Current()}
	switch m.updateStage {
	case updateChecking:
		update.label = "Checking for updates…"
		update.desc = ""
	case updateAvailable:
		if m.updateRel != nil {
			update.label = "Update to " + m.updateRel.Version
			update.desc = "restart applies it"
		}
	case updateInstalling:
		update.label = "Downloading update…"
		update.desc = ""
	case updateCurrent:
		update.desc = "up to date · " + version.Current()
	case updateDone:
		update.label = "Update installed"
		update.desc = "restart to apply"
	}
	return []menuRow{
		update,
		incognitoRow(),
		{kind: rowAction, id: "rename", label: "Rename device"},
		{kind: rowAction, id: "quit", label: "Quit"},
	}
}

func incognitoRow() menuRow {
	p := privacy.Shared().Policy()
	row := menuRow{kind: rowCheck, id: "incognito", label: "Incognito", desc: "hide what you're doing", checked: p.Hidden()}
	switch {
	case !p.Hidden():
	case p.Note != "":
		row.desc = p.Note
	default:
		row.desc = "the room sees a quiet card"
	}
	if until, ok := p.Expires(); ok {
		row.desc = "back at " + until.Format("15:04")
	}
	return row
}

func (m model) panelRows() []menuRow {
	return []menuRow{
		{kind: rowAction, id: "chat", label: "Chat", desc: "room messages", checked: m.panel == panelChat},
		{kind: rowAction, id: "board", label: "Screen today", desc: "who's been at the screen", checked: m.panel == panelBoard},
	}
}

func (m model) menuPopup(width, height int) string {
	rows := m.menuRows()
	inner := max(min(width/2, 52), 30)
	textW := max(inner-popupBox.GetHorizontalPadding(), 16)

	lines := []string{modalTitle.Render(m.menuTitle()), ""}
	for i, r := range rows {
		cursor := "  "
		if i == m.menuIndex {
			cursor = accentStyle.Render("▸ ")
		}

		var label string
		switch r.kind {
		case rowSection:
			marker := "▸"
			if r.open {
				marker = "▾"
			}
			label = accentStyle.Render(marker+" ") + accentStyle.Render(r.label)
		case rowCheck:
			label = "  " + checkbox(r.checked) + styleFor(i == m.menuIndex).Render(r.label)
		default:
			label = "  " + styleFor(i == m.menuIndex).Render(r.label)
		}

		row := cursor + label
		if r.desc != "" {
			row += dimStyle.Render("  " + r.desc)
		}
		lines = append(lines, ansi.Truncate(row, textW, "…"))
	}

	if maxRows := max(height-8, 6); len(lines) > maxRows {
		lines = lines[:maxRows]
	}
	return popupBox.Width(inner).Render(strings.Join(lines, "\n"))
}

func styleFor(selected bool) interface{ Render(...string) string } {
	if selected {
		return accentStyle
	}
	return dimStyle
}

func (m model) menuFooter() string {
	rows := m.menuRows()
	if m.menuIndex >= 0 && m.menuIndex < len(rows) {
		switch rows[m.menuIndex].kind {
		case rowSection:
			key := accentStyle.Render("→")
			word := " open · "
			if rows[m.menuIndex].open {
				key, word = accentStyle.Render("←"), " close · "
			}
			return dimStyle.Render("↑↓ · ") + key + dimStyle.Render(word) +
				accentStyle.Render("esc") + dimStyle.Render(" room")
		case rowCheck:
			return dimStyle.Render("↑↓ · ") + accentStyle.Render("enter") + dimStyle.Render(" toggle · ") +
				accentStyle.Render("←") + dimStyle.Render(" collapse · ") + accentStyle.Render("esc") + dimStyle.Render(" room")
		}
	}
	return dimStyle.Render("↑↓ · ") + accentStyle.Render("enter") + dimStyle.Render(" select · ") +
		accentStyle.Render("esc") + dimStyle.Render(" room")
}
