package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"statusphere-client/internal/presence"
)

type sentMsg struct{ to, text string }

type recordingCtrl struct {
	sent   []sentMsg
	kicked []string
	rendev []string
}

func (c *recordingCtrl) SendMessage(to, text string) { c.sent = append(c.sent, sentMsg{to, text}) }
func (c *recordingCtrl) Kick(id string)              { c.kicked = append(c.kicked, id) }
func (c *recordingCtrl) Rename(n string)             { c.rendev = append(c.rendev, n) }
func (c *recordingCtrl) SyncSpotify(string)          {}
func (c *recordingCtrl) SyncCustom([]string)         {}

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func send(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func TestTabFocusesChatAndSends(t *testing.T) {
	ctrl := &recordingCtrl{}
	m := newModel(Options{LocalAccountID: meAccount, Controller: ctrl})
	m.width, m.height = 90, 30
	m.groups = groupDevices([]presence.Snapshot{
		{presence.KeyDeviceID: "me-1", presence.KeyAccountID: meAccount, presence.KeyLastSeen: int64(0)},
	}, meAccount)

	m = send(m, key("tab"))
	if m.focus != focusChat {
		t.Fatal("tab from cards should focus the chat")
	}

	for _, ch := range []string{"h", "i"} {
		m = send(m, key(ch))
	}
	if m.chatInput != "hi" {
		t.Fatalf("typing should build the chat draft, got %q", m.chatInput)
	}

	m = send(m, key("enter"))
	if len(ctrl.sent) != 1 || ctrl.sent[0].to != "" || ctrl.sent[0].text != "hi" {
		t.Fatalf("enter should send a group message, got %+v", ctrl.sent)
	}
	if m.chatInput != "" {
		t.Fatal("chat draft should clear after send")
	}

	m = send(m, key("esc"))
	if m.focus != focusCards {
		t.Fatal("esc from chat should return focus to cards")
	}
}

func TestQTypedInChatDoesNotQuit(t *testing.T) {
	ctrl := &recordingCtrl{}
	m := newModel(Options{LocalAccountID: meAccount, Controller: ctrl})
	m.width, m.height = 90, 30
	m.focus = focusChat

	next, cmd := m.Update(key("q"))
	m = next.(model)
	if cmd != nil {
		t.Fatal("q while typing in chat must not quit")
	}
	if m.chatInput != "q" {
		t.Fatalf("q should be typed into the chat draft, got %q", m.chatInput)
	}
}
