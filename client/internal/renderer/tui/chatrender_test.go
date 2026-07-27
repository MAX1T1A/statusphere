package tui

import (
	"strings"
	"testing"

	"statusphere-client/internal/presence"
)

func chatModel() model {
	m := newModel(Options{LocalAccountID: meAccount})
	m.width, m.height = 90, 30
	m.groups = groupDevices([]presence.Snapshot{
		{presence.KeyDeviceID: "bob-1", presence.KeyAccountID: "acc-bob", presence.KeyAccountName: "Bob", presence.KeyLastSeen: int64(0)},
		{presence.KeyDeviceID: "me-1", presence.KeyAccountID: meAccount, presence.KeyAccountName: "Me", presence.KeyLastSeen: int64(0)},
	}, meAccount)
	return m
}

func TestGroupChatPanelRenders(t *testing.T) {
	m := chatModel()
	m.chat.Ingest("acc-bob", "Bob", "", "hey everyone", "")
	m.chat.Ingest(meAccount, "Me", "", "hi bob", "")

	out := m.groupChatPanel(40, 16)
	for _, want := range []string{"group chat", "hey everyone", "hi bob", "Bob", "you"} {
		if !strings.Contains(out, want) {
			t.Fatalf("group chat panel missing %q\n%s", want, out)
		}
	}
}

func TestSelfCardPinnedFirstWithDivider(t *testing.T) {
	m := chatModel()
	if m.groups[0].key != meAccount {
		t.Fatalf("own card must be pinned first, got %q", m.groups[0].key)
	}
	out, _ := m.renderCards(60, 40)
	if !strings.Contains(out, "members") {
		t.Fatalf("expected a divider before the other members:\n%s", out)
	}
}

func TestDMModalRendersAndResolvesNameFromHistory(t *testing.T) {
	m := chatModel()
	m.chat.Ingest("acc-bob", "", meAccount, "psst just you", "")
	m.mode = modeDM
	m.dmPeer = "acc-bob"
	m.dmPeerName = "Bob"

	out := m.View()
	for _, want := range []string{"Bob", "direct message", "psst just you"} {
		if !strings.Contains(out, want) {
			t.Fatalf("dm modal missing %q\n%s", want, out)
		}
	}
}

func TestMenuIncludesMessageForOtherNotSelf(t *testing.T) {
	m := chatModel()

	selectAccount(&m, "acc-bob")
	if !hasAction(m.personRows(), "message") {
		t.Fatal("expected Message item when focused on another account")
	}

	selectAccount(&m, meAccount)
	if hasAction(m.personRows(), "message") {
		t.Fatal("Message item must be hidden when focused on self")
	}
}

func hasAction(rows []menuRow, action string) bool {
	for _, r := range rows {
		if r.id == action {
			return true
		}
	}
	return false
}
