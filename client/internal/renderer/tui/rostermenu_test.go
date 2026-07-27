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
	})
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

func TestOwnerSeesKickForMemberOnly(t *testing.T) {
	m := rosterModel("owner")

	selectAccount(&m, "acc-bob") // another member
	if !hasAction(m.menu(), "kick") {
		t.Fatal("owner should see Remove for a member")
	}
	if !hasAction(m.menu(), "message") {
		t.Fatal("owner should also see Message for a member")
	}

	selectAccount(&m, meAccount) // self
	if hasAction(m.menu(), "kick") || hasAction(m.menu(), "message") {
		t.Fatal("no kick/message on your own card")
	}

	selectAccount(&m, "acc-ann") // offline member is still kickable
	if !hasAction(m.menu(), "kick") {
		t.Fatal("owner should be able to remove an offline member")
	}
}

func TestNonOwnerNeverSeesKick(t *testing.T) {
	m := rosterModel("member") // I am not the owner
	selectAccount(&m, "acc-bob")
	if hasAction(m.menu(), "kick") {
		t.Fatal("a non-owner must never see Remove from room")
	}
	if !hasAction(m.menu(), "message") {
		t.Fatal("a non-owner should still be able to DM a member")
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
