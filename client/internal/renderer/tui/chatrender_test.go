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
	})
	return m
}

func TestGroupModalRenders(t *testing.T) {
	m := chatModel()
	m.chat.Ingest("acc-bob", "Bob", "", "hey everyone", "")
	m.chat.Ingest(meAccount, "Me", "", "hi bob", "")
	m.mode = modeChat

	out := m.View()
	for _, want := range []string{"group chat", "hey everyone", "hi bob", "Bob", "you"} {
		if !strings.Contains(out, want) {
			t.Fatalf("group modal missing %q\n%s", want, out)
		}
	}
}

func TestDMModalRendersAndResolvesNameFromHistory(t *testing.T) {
	m := chatModel()
	// history has no from_name; name must resolve from the room roster
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

	m.selected = 0 // Bob
	if !hasAction(m.menu(), "message") {
		t.Fatal("expected Message item when focused on another account")
	}

	m.selected = 1 // Me
	if hasAction(m.menu(), "message") {
		t.Fatal("Message item must be hidden when focused on self")
	}
}

func hasAction(items []menuItem, action string) bool {
	for _, it := range items {
		if it.action == action {
			return true
		}
	}
	return false
}
