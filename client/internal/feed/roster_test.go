package feed

import (
	"testing"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/presence"
)

func rosterOf(members ...auth.MemberInfo) *Roster {
	r := NewRoster(nil)
	r.members = members
	return r
}

func TestRosterFallbackToLiveBeforeMembers(t *testing.T) {
	f := New()
	f.Update(presence.Snapshot{presence.KeyDeviceID: "d1", presence.KeyAccountID: "acc-bob"})

	got := rosterOf().Merge(f.Snapshot())
	if len(got) != 1 {
		t.Fatalf("before member fetch roster should fall back to live: want 1, got %d", len(got))
	}
}

func TestRosterKeepsOfflineMembersDropsNonMembers(t *testing.T) {
	f := New()
	f.Update(presence.Snapshot{presence.KeyDeviceID: "bob-1", presence.KeyAccountID: "acc-bob"})
	f.Update(presence.Snapshot{presence.KeyDeviceID: "eve-1", presence.KeyAccountID: "acc-eve"})
	r := rosterOf(
		auth.MemberInfo{AccountID: "acc-bob", Name: "Bob", Role: "owner"},
		auth.MemberInfo{AccountID: "acc-ann", Name: "Ann", Role: "member"},
	)

	byAcc := map[string]presence.Snapshot{}
	for _, s := range r.Merge(f.Snapshot()) {
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
	f := New()
	f.UpdateOwn(presence.Snapshot{}, "me-dev", "acc-me", "")
	r := rosterOf(auth.MemberInfo{AccountID: "acc-me", Name: "Me", Role: "owner"})

	got := r.Merge(f.Snapshot())
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
	r := rosterOf(auth.MemberInfo{AccountID: "0123456789abcdef", Name: "", Role: "member"})

	got := r.Merge(New().Snapshot())
	if len(got) != 1 || got[0].String(presence.KeyAccountName) != "01234567" {
		t.Fatalf("nameless offline member should label with short id, got %+v", got)
	}
}

func TestRosterOfflineLabelKeepsLastSeenName(t *testing.T) {
	r := rosterOf(auth.MemberInfo{AccountID: "0123456789abcdef", Role: "member"})
	f := New()
	f.Update(presence.Snapshot{
		presence.KeyDeviceID:   "d1",
		presence.KeyAccountID:  "0123456789abcdef",
		presence.KeyDeviceName: "thinkpad",
	})
	r.Merge(f.Snapshot())

	got := r.Merge(New().Snapshot())
	if len(got) != 1 || got[0].String(presence.KeyAccountName) != "thinkpad" {
		t.Fatalf("offline card should keep the name seen while online, got %+v", got)
	}
}

func TestRosterSeenSignalsOnUnknown(t *testing.T) {
	r := rosterOf(auth.MemberInfo{AccountID: "acc-bob"})

	r.Seen("acc-bob")
	select {
	case <-r.refresh:
		t.Fatal("known account should not trigger a refresh")
	default:
	}

	r.Seen("acc-new")
	select {
	case <-r.refresh:
	default:
		t.Fatal("unknown account should trigger a refresh")
	}
}
