package presence_test

import (
	"encoding/json"
	"testing"

	"statusphere-client/internal/presence"
)

func TestFloatHandlesNumericKinds(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want float64
	}{
		{"float64", float64(1.5), 1.5},
		{"int", int(3), 3},
		{"int64", int64(7), 7},
		{"json.Number", json.Number("2.25"), 2.25},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := presence.Snapshot{"k": c.val}
			got, ok := s.Float("k")
			if !ok || got != c.want {
				t.Fatalf("Float(%v) = %v, %v; want %v", c.val, got, ok, c.want)
			}
		})
	}
	if _, ok := (presence.Snapshot{"k": "x"}).Float("k"); ok {
		t.Fatal("Float on string should be false")
	}
}

func TestIntHandlesJSONRoundTrip(t *testing.T) {
	orig := presence.Snapshot{presence.KeyLastSeen: int64(1700000000)}
	data, err := json.Marshal(map[string]any(orig))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	s := presence.Snapshot(decoded)
	got, ok := s.Int(presence.KeyLastSeen)
	if !ok || got != 1700000000 {
		t.Fatalf("after round trip Int = %v, %v; want 1700000000", got, ok)
	}
}

func TestStrings(t *testing.T) {
	if got := (presence.Snapshot{"k": []string{"a", "b"}}).Strings("k"); len(got) != 2 {
		t.Fatalf("[]string: got %v", got)
	}
	got := (presence.Snapshot{"k": []any{"a", "", "b", 3}}).Strings("k")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("[]any filtered: got %v", got)
	}
}

func TestEqual(t *testing.T) {
	a := presence.Snapshot{"x": 1.0, "s": "hi"}
	b := presence.Snapshot{"x": 1, "s": "hi"}
	if !a.Equal(b) {
		t.Fatal("int 1 and float 1.0 should be Equal")
	}
	if a.Equal(presence.Snapshot{"x": 1.0}) {
		t.Fatal("different length should not be Equal")
	}
	if a.Equal(presence.Snapshot{"x": 1.0, "s": "bye"}) {
		t.Fatal("different value should not be Equal")
	}
}

func TestEqualExceptIgnoresVolatile(t *testing.T) {
	ignore := map[string]bool{presence.KeyUptimeHours: true}
	a := presence.Snapshot{presence.KeyUptimeHours: 1.0, presence.KeyActiveApp: "kitty"}
	b := presence.Snapshot{presence.KeyUptimeHours: 99.0, presence.KeyActiveApp: "kitty"}
	if !a.EqualExcept(b, ignore) {
		t.Fatal("differing only in ignored key should be equal")
	}
	c := presence.Snapshot{presence.KeyUptimeHours: 1.0, presence.KeyActiveApp: "firefox"}
	if a.EqualExcept(c, ignore) {
		t.Fatal("differing in non-ignored key should not be equal")
	}
	d := presence.Snapshot{presence.KeyUptimeHours: 1.0, presence.KeyActiveApp: "kitty", presence.KeySpotifyURI: "u"}
	if a.EqualExcept(d, ignore) {
		t.Fatal("extra non-ignored key should not be equal")
	}
}

func TestCloneIsIndependent(t *testing.T) {
	a := presence.Snapshot{"x": 1}
	b := a.Clone()
	b["x"] = 2
	b["y"] = 3
	if v, _ := a.Int("x"); v != 1 {
		t.Fatal("clone mutation leaked into original")
	}
	if a.Has("y") {
		t.Fatal("clone new key leaked into original")
	}
}
