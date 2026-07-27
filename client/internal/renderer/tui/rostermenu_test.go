package tui

import (
	"strings"
	"testing"

	"statusphere-client/internal/presence"
)

func selectAccount(m *model, acc string) {
	for i, g := range m.groups {
		if g.key == acc {
			m.selected = i
			return
		}
	}
}

func rosterModel(myRole string) model {
	m := newModel(Options{LocalAccountID: meAccount})
	m.width, m.height = 90, 30
	m.groups = groupDevices([]presence.Snapshot{
		{presence.KeyDeviceID: "me-1", presence.KeyAccountID: meAccount, presence.KeyAccountName: "Me", presence.KeyRole: myRole, presence.KeyLastSeen: int64(0)},
		{presence.KeyDeviceID: "bob-1", presence.KeyAccountID: "acc-bob", presence.KeyAccountName: "Bob", presence.KeyRole: "member", presence.KeyLastSeen: int64(0)},
		{presence.KeyAccountID: "acc-ann", presence.KeyAccountName: "Ann", presence.KeyRole: "member", presence.KeyOffline: true},
	}, meAccount)
	return m
}

func TestOfflineMemberCardRenders(t *testing.T) {
	m := rosterModel("owner")
	selectAccount(&m, "acc-ann")
	out := renderCard(m.groups[m.selected], m.blocks, m.custom, true, 60, false)
	if !strings.Contains(out, "Ann") || !strings.Contains(out, "offline") {
		t.Fatalf("offline member card should show name + offline:\n%s", out)
	}
}

func TestPersonMenuHasNoKickOrSettings(t *testing.T) {
	m := rosterModel("owner")
	selectAccount(&m, "acc-bob")
	if hasAction(m.personMenu(), "kick") || hasAction(m.personMenu(), "rename") || hasAction(m.personMenu(), "quit") {
		t.Fatal("person menu must only hold per-person actions (music/screen/message)")
	}
	if !hasAction(m.personMenu(), "message") {
		t.Fatal("person menu should offer Message for another member")
	}
}

func TestOwnerKicksWithXAndConfirm(t *testing.T) {
	ctrl := &recordingCtrl{}
	m := rosterModel("owner")
	m.ctrl = ctrl
	selectAccount(&m, "acc-bob")

	m = send(m, key("x"))
	if m.confirmKick != "acc-bob" {
		t.Fatalf("x should arm a kick confirmation, got %q", m.confirmKick)
	}
	m = send(m, key("y"))
	if len(ctrl.kicked) != 1 || ctrl.kicked[0] != "acc-bob" {
		t.Fatalf("y should confirm the kick, got %v", ctrl.kicked)
	}
	if m.confirmKick != "" {
		t.Fatal("confirmation should clear after y")
	}
}

func TestKickConfirmCancels(t *testing.T) {
	ctrl := &recordingCtrl{}
	m := rosterModel("owner")
	m.ctrl = ctrl
	selectAccount(&m, "acc-bob")
	m = send(m, key("x"))
	m = send(m, key("n"))
	if len(ctrl.kicked) != 0 || m.confirmKick != "" {
		t.Fatalf("n should cancel the kick, kicked=%v confirm=%q", ctrl.kicked, m.confirmKick)
	}
}

func TestNonOwnerCannotKick(t *testing.T) {
	ctrl := &recordingCtrl{}
	m := rosterModel("member") // not the owner
	m.ctrl = ctrl
	selectAccount(&m, "acc-bob")
	m = send(m, key("x"))
	if m.confirmKick != "" {
		t.Fatal("a non-owner pressing x must not arm a kick")
	}
}

func TestSettingsOpensAndRenames(t *testing.T) {
	m := rosterModel("member")
	m = send(m, key("s"))
	if m.mode != modeSettings {
		t.Fatalf("s should open settings, mode=%v", m.mode)
	}
	items, _ := m.currentMenu()
	if !hasAction(items, "rename") || !hasAction(items, "quit") {
		t.Fatalf("settings should hold rename + quit, got %v", items)
	}
	// rename is the first item; enter opens the rename input
	m.menuIndex = 0
	next, _ := m.runMenu()
	if next.(model).mode != modeRename {
		t.Fatal("selecting Rename device should open the rename input")
	}
}

func TestSelfCardNotSelectable(t *testing.T) {
	m := rosterModel("owner") // groups: [me(self), bob, ann]
	// selection starts at the first non-self member and up-arrow cannot reach self
	m.clampSelection()
	if m.selected != 1 {
		t.Fatalf("selection should skip the pinned self card, got %d", m.selected)
	}
	m = send(m, key("up"))
	if m.selected == 0 {
		t.Fatal("up must not land on your own (static) card")
	}
	if m.focusedDevice().String(presence.KeyAccountID) == meAccount {
		t.Fatal("your own card must never be the focused/selected member")
	}
}

func TestIsOwner(t *testing.T) {
	if !rosterModel("owner").isOwner() {
		t.Fatal("expected owner")
	}
	if rosterModel("member").isOwner() {
		t.Fatal("expected non-owner")
	}
}
