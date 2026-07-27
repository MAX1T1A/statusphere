package tui

import (
	"testing"
)

func TestPanelViewPersistsRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if got := loadPanelView(); got != panelChat {
		t.Fatalf("default panel should be chat, got %v", got)
	}
	savePanelView(panelBoard)
	if got := loadPanelView(); got != panelBoard {
		t.Fatal("board choice should survive a reload")
	}
	savePanelView(panelChat)
	if got := loadPanelView(); got != panelChat {
		t.Fatal("chat choice should survive a reload")
	}
}

func TestSwitchingPanelSavesChoice(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := chatModel()
	m = send(m, key("v"))
	m = send(m, key("down"))
	next, _ := m.runMenu()
	m = next.(model)

	if m.panel != panelBoard || loadPanelView() != panelBoard {
		t.Fatalf("picking Screen today should persist, panel=%v saved=%v", m.panel, loadPanelView())
	}
}

func TestGroupUnreadClearsWhenChatVisible(t *testing.T) {
	m := chatModel()
	m.chat.Ingest("acc-bob", "Bob", "", "ping", "")
	if m.chat.GroupUnread() != 1 {
		t.Fatal("expected 1 unread")
	}

	// chat panel is on the main screen -> auto-read on the next feed tick
	m.panel = panelChat
	m = send(m, FeedMsg{})
	if m.chat.GroupUnread() != 0 {
		t.Fatal("visible chat panel should clear group unread")
	}
}

func TestGroupUnreadKeptWhileBoardActive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := chatModel()
	m.panel = panelBoard
	m.chat.Ingest("acc-bob", "Bob", "", "ping", "")
	m = send(m, FeedMsg{})
	if m.chat.GroupUnread() != 1 {
		t.Fatal("board view must keep the unread bell")
	}

	// switching back to chat via the picker clears it instantly
	m = send(m, key("v"))
	m.menuIndex = 0
	next, _ := m.runMenu()
	m = next.(model)
	if m.chat.GroupUnread() != 0 {
		t.Fatal("switching to chat view should mark group read")
	}
}
