package tui

import (
	"testing"

	"statusphere-client/internal/chat"
)

const meAccount = "acc-me"

func TestIngestGroupFromOther(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest("acc-bob", "Bob", "", "hello room", "")

	if got := c.GroupUnread(); got != 1 {
		t.Fatalf("group unread = %d, want 1", got)
	}
	entries := c.GroupEntries()
	if len(entries) != 1 || entries[0].Self || entries[0].Text != "hello room" || entries[0].Name != "Bob" {
		t.Fatalf("unexpected group entries: %+v", entries)
	}
	if c.DMUnread("acc-bob") != 0 {
		t.Fatal("group message must not create DM unread")
	}
}

func TestIngestGroupSelfEchoNoUnread(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest(meAccount, "Me", "", "my own line", "")

	if got := c.GroupUnread(); got != 0 {
		t.Fatalf("self echo bumped group unread to %d", got)
	}
	entries := c.GroupEntries()
	if len(entries) != 1 || !entries[0].Self {
		t.Fatalf("self echo not marked self: %+v", entries)
	}
}

func TestIngestIncomingDM(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest("acc-bob", "Bob", meAccount, "just you", "")

	if got := c.DMUnread("acc-bob"); got != 1 {
		t.Fatalf("dm unread(bob) = %d, want 1", got)
	}
	if c.GroupUnread() != 0 {
		t.Fatal("DM must not bump group unread")
	}
	entries := c.DMEntries("acc-bob")
	if len(entries) != 1 || entries[0].Self || entries[0].Text != "just you" {
		t.Fatalf("unexpected dm entries: %+v", entries)
	}
}

func TestIngestOwnDMEchoKeyedByPeerNoUnread(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest(meAccount, "Me", "acc-bob", "reply", "")

	if got := c.DMUnread("acc-bob"); got != 0 {
		t.Fatalf("own dm echo bumped unread to %d", got)
	}
	entries := c.DMEntries("acc-bob")
	if len(entries) != 1 || !entries[0].Self || entries[0].Text != "reply" {
		t.Fatalf("own dm echo not keyed/marked correctly: %+v", entries)
	}
}

func TestDMThreadsAreSeparatePerPeer(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest("acc-bob", "Bob", meAccount, "from bob", "")
	c.Ingest("acc-ann", "Ann", meAccount, "from ann", "")

	if c.DMUnread("acc-bob") != 1 || c.DMUnread("acc-ann") != 1 {
		t.Fatal("per-peer unread counts wrong")
	}
	if len(c.DMEntries("acc-bob")) != 1 || len(c.DMEntries("acc-ann")) != 1 {
		t.Fatal("threads leaked across peers")
	}
}

func TestMarkRead(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest("acc-bob", "Bob", "", "g", "")
	c.Ingest("acc-bob", "Bob", meAccount, "d", "")

	c.MarkGroupRead()
	c.MarkDMRead("acc-bob")
	if c.GroupUnread() != 0 || c.DMUnread("acc-bob") != 0 {
		t.Fatal("mark read did not clear unread")
	}
}

func TestEmptyTextIgnored(t *testing.T) {
	c := NewChatStore(meAccount)
	c.Ingest("acc-bob", "Bob", "", "", "")
	if len(c.GroupEntries()) != 0 || c.GroupUnread() != 0 {
		t.Fatal("empty message was stored")
	}
}

func TestLoadHistoryRoutesWithoutUnread(t *testing.T) {
	c := NewChatStore(meAccount)
	c.LoadHistory([]chat.Message{
		{From: "acc-bob", To: "", Text: "old group", At: ""},
		{From: "acc-bob", To: meAccount, Text: "old dm in", At: ""},
		{From: meAccount, To: "acc-bob", Text: "old dm out", At: ""},
		{From: meAccount, To: "", Text: "old group me", At: ""},
	})

	if c.GroupUnread() != 0 || c.DMUnread("acc-bob") != 0 {
		t.Fatal("history load must not create unread")
	}
	if len(c.GroupEntries()) != 2 {
		t.Fatalf("group history = %d, want 2", len(c.GroupEntries()))
	}
	dm := c.DMEntries("acc-bob")
	if len(dm) != 2 {
		t.Fatalf("dm history = %d, want 2", len(dm))
	}
	// order preserved as supplied; self flag derived from sender
	if dm[0].Self || !dm[1].Self {
		t.Fatalf("history self flags wrong: %+v", dm)
	}
}

func TestLoadHistoryOrdersAndDedupsAgainstLive(t *testing.T) {
	c := NewChatStore(meAccount)
	// a live message lands before history finishes loading
	c.Ingest("acc-bob", "Bob", "", "live newest", "2026-07-26T10:00:05Z")
	// history (older) plus a duplicate of the live message (same author/text/timestamp)
	c.LoadHistory([]chat.Message{
		{From: "acc-bob", To: "", Text: "old one", At: "2026-07-26T10:00:01Z"},
		{From: "acc-bob", To: "", Text: "old two", At: "2026-07-26T10:00:02Z"},
		{From: "acc-bob", To: "", Text: "live newest", At: "2026-07-26T10:00:05Z"},
	})

	entries := c.GroupEntries()
	texts := make([]string, len(entries))
	for i, e := range entries {
		texts[i] = e.Text
	}
	want := []string{"old one", "old two", "live newest"}
	if len(texts) != len(want) {
		t.Fatalf("got %v, want %v (dup not removed or order wrong)", texts, want)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Fatalf("order/dedup wrong: got %v, want %v", texts, want)
		}
	}
}

func TestChatMaxTrim(t *testing.T) {
	c := NewChatStore(meAccount)
	for i := 0; i < chatMax+50; i++ {
		c.Ingest("acc-bob", "Bob", "", "spam", "")
	}
	if got := len(c.GroupEntries()); got != chatMax {
		t.Fatalf("group log not trimmed to %d, got %d", chatMax, got)
	}
}
