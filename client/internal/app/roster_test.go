package app

import (
	"testing"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/feed"
	"statusphere-client/internal/presence"
)

func TestRosterFallbackToLiveBeforeMembers(t *testing.T) {
	a := &App{feed: feed.New()}
	a.feed.Update(presence.Snapshot{presence.KeyDeviceID: "d1", presence.KeyAccountID: "acc-bob"})

	got := a.roster()
	if len(got) != 1 {
		t.Fatalf("before member fetch roster should fall back to live: want 1, got %d", len(got))
	}
}

func TestRosterKeepsOfflineMembersDropsNonMembers(t *testing.T) {
	a := &App{feed: feed.New()}
	a.feed.Update(presence.Snapshot{presence.KeyDeviceID: "bob-1", presence.KeyAccountID: "acc-bob"})
	a.feed.Update(presence.Snapshot{presence.KeyDeviceID: "eve-1", presence.KeyAccountID: "acc-eve"})
	a.members = []auth.MemberInfo{
		{AccountID: "acc-bob", Name: "Bob", Role: "owner"},
		{AccountID: "acc-ann", Name: "Ann", Role: "member"},
	}

	byAcc := map[string]presence.Snapshot{}
	for _, s := range a.roster() {
		byAcc[s.String(presence.KeyAccountID)] = s
	}

	if len(byAcc) != 2 {
		t.Fatalf("want exactly bob + ann, got %d: %v", len(byAcc), byAcc)
	}
	if _, ok := byAcc["acc-eve"]; ok {
		t.Fatal("a live account not in the member list (kicked) must be dropped")
	}
	bob := byAcc["acc-bob"]
	if bob.Has(presence.KeyOffline) || bob.String(presence.KeyRole) != "owner" {
		t.Fatalf("bob should be online with owner role: %+v", bob)
	}
	ann := byAcc["acc-ann"]
	if !ann.Has(presence.KeyOffline) || ann.String(presence.KeyAccountName) != "Ann" || ann.String(presence.KeyRole) != "member" {
		t.Fatalf("ann should be an offline placeholder named Ann: %+v", ann)
	}
}

func TestRosterSelfIsOnlineWithNameFromMembers(t *testing.T) {
	a := &App{feed: feed.New()}
	a.feed.Update(presence.Snapshot{presence.KeyDeviceID: "me-dev", presence.KeyAccountID: "acc-me"})
	a.members = []auth.MemberInfo{{AccountID: "acc-me", Name: "Me", Role: "owner"}}

	got := a.roster()
	if len(got) != 1 {
		t.Fatalf("want just self, got %d", len(got))
	}
	self := got[0]
	if self.Has(presence.KeyOffline) {
		t.Fatal("local user must not render as offline")
	}
	if self.String(presence.KeyAccountName) != "Me" || self.String(presence.KeyRole) != "owner" {
		t.Fatalf("self should carry name+role from members: %+v", self)
	}
}

func TestRosterOfflineLabelFallsBackToShortID(t *testing.T) {
	a := &App{feed: feed.New()}
	a.members = []auth.MemberInfo{{AccountID: "0123456789abcdef", Name: "", Role: "member"}}

	got := a.roster()
	if len(got) != 1 || got[0].String(presence.KeyAccountName) != "01234567" {
		t.Fatalf("nameless offline member should label with short id, got %+v", got)
	}
}

func TestRosterOfflineLabelKeepsLastSeenName(t *testing.T) {
	a := &App{feed: feed.New()}
	a.members = []auth.MemberInfo{{AccountID: "0123456789abcdef", Role: "member"}}
	a.feed.Update(presence.Snapshot{
		presence.KeyDeviceID:   "d1",
		presence.KeyAccountID:  "0123456789abcdef",
		presence.KeyDeviceName: "thinkpad",
	})
	a.roster()

	a.feed = feed.New() // the device aged out of the feed
	got := a.roster()
	if len(got) != 1 || got[0].String(presence.KeyAccountName) != "thinkpad" {
		t.Fatalf("offline card should keep the name seen while online, got %+v", got)
	}
}

func TestMaybeRefreshMembersSignalsOnUnknown(t *testing.T) {
	a := &App{memberRefresh: make(chan struct{}, 1)}
	a.members = []auth.MemberInfo{{AccountID: "acc-bob"}}

	a.maybeRefreshMembers("acc-bob")
	select {
	case <-a.memberRefresh:
		t.Fatal("known account should not trigger a refresh")
	default:
	}

	a.maybeRefreshMembers("acc-new")
	select {
	case <-a.memberRefresh:
	default:
		t.Fatal("unknown account should trigger a refresh")
	}
}
