package privacy

import (
	"fmt"
	"testing"
	"time"

	"statusphere-client/internal/presence"
)

func sample() presence.Snapshot {
	return presence.Snapshot{
		presence.KeyActiveApp:       "firefox",
		presence.KeyActiveWindow:    "tax return 2025 — Mozilla Firefox",
		presence.KeyActiveWorkspace: "3",
		presence.KeySpotifyStatus:   "playing",
		presence.KeySpotifyTrack:    "Teardrop",
		presence.KeySpotifyArtist:   "Massive Attack",
		presence.KeyMusic:           "How to file a tax return - YouTube",
		presence.KeyCPUPercent:      12.5,
		presence.KeyCustomFields:    []string{"weather"},
		"weather":                   "18°C",
	}
}

func policy(prof Profile) Policy {
	p := Default()
	p.Mode = ModeIncognito
	p.Profiles[ModeIncognito] = prof
	return p
}

func TestIncognitoDropsAppsKeepsMusic(t *testing.T) {
	p := Default()
	p.Mode = ModeIncognito
	out := New(p).Apply(sample())

	for _, key := range appKeys {
		if out.Has(key) {
			t.Errorf("hidden snapshot still carries %s", key)
		}
	}
	if out.String(presence.KeySpotifyTrack) != "Teardrop" {
		t.Error("incognito should keep music: it's the sociable part")
	}
	if !out.Has(presence.KeyIncognito) {
		t.Error("announce is on by default: a quiet card beats a frozen one")
	}
	if New(Default()).Apply(sample()).Has(presence.KeyIncognito) {
		t.Error("normal mode must not announce")
	}
}

func TestApplyDoesNotMutateInput(t *testing.T) {
	in := sample()
	New(policy(Profile{Apps: LevelOff})).Apply(in)
	if !in.Has(presence.KeyActiveWindow) {
		t.Error("filter mutated the caller's snapshot")
	}
}

func TestLevels(t *testing.T) {
	cases := []struct {
		name string
		prof Profile
		want func(presence.Snapshot) error
	}{
		{"apps app keeps name drops title", Profile{Apps: LevelApp}, func(s presence.Snapshot) error {
			if s.String(presence.KeyActiveApp) != "firefox" || s.Has(presence.KeyActiveWindow) {
				return fmt.Errorf("got app=%q window=%v", s.String(presence.KeyActiveApp), s.Has(presence.KeyActiveWindow))
			}
			return nil
		}},
		{"apps busy replaces name", Profile{Apps: LevelBusy}, func(s presence.Snapshot) error {
			if s.String(presence.KeyActiveApp) != BusyLabel || s.Has(presence.KeyActiveWorkspace) {
				return fmt.Errorf("got app=%q workspace=%v", s.String(presence.KeyActiveApp), s.Has(presence.KeyActiveWorkspace))
			}
			return nil
		}},
		{"music listening keeps status only", Profile{Music: LevelListening}, func(s presence.Snapshot) error {
			if s.String(presence.KeySpotifyStatus) != "playing" || s.Has(presence.KeySpotifyTrack) {
				return fmt.Errorf("got status=%q track=%v", s.String(presence.KeySpotifyStatus), s.Has(presence.KeySpotifyTrack))
			}
			return nil
		}},
		{"system off drops metrics", Profile{System: LevelOff}, func(s presence.Snapshot) error {
			if s.Has(presence.KeyCPUPercent) {
				return fmt.Errorf("cpu survived")
			}
			return nil
		}},
		{"custom off drops the fields it lists", Profile{Custom: LevelOff}, func(s presence.Snapshot) error {
			if s.Has("weather") || s.Has(presence.KeyCustomFields) {
				return fmt.Errorf("custom fields survived")
			}
			return nil
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.want(New(policy(c.prof)).Apply(sample())); err != nil {
				t.Error(err)
			}
		})
	}
}

// The generic mpris title carries browser video names, so hiding the apps has
// to take it along or incognito leaks what you're watching.
func TestGenericMusicFollowsTheApps(t *testing.T) {
	p := Default()
	p.Mode = ModeIncognito

	out := New(p).Apply(sample())
	if out.Has(presence.KeyMusic) {
		t.Error("generic now-playing survived incognito")
	}
	if out.String(presence.KeySpotifyTrack) != "Teardrop" {
		t.Error("spotify metadata should still be there")
	}
	if New(Default()).Apply(sample()).String(presence.KeyMusic) == "" {
		t.Error("visible mode should publish it as before")
	}
}

// A field no collector version knows about must not walk out while hidden.
func TestUnknownKeysDroppedWhileHidden(t *testing.T) {
	p := Default()
	p.Mode = ModeIncognito

	in := sample()
	in.Set("clipboard_contents", "hunter2")

	if New(p).Apply(in).Has("clipboard_contents") {
		t.Error("unclassified field survived incognito")
	}
	if !New(p).Apply(in).Has("weather") {
		t.Error("a field declared in custom_fields should follow the custom level")
	}
}

func TestHideAppsAppliesInNormalMode(t *testing.T) {
	p := Default()
	p.HideApps = []string{"(?i)tax return"}

	out := New(p).Apply(sample())
	if out.Has(presence.KeyActiveApp) || out.Has(presence.KeyActiveWindow) {
		t.Error("a matching window must be hidden even while visible")
	}
	if out.String(presence.KeySpotifyTrack) != "Teardrop" {
		t.Error("hide_apps should only touch the app group")
	}
}

func TestAnnounce(t *testing.T) {
	p := Default()
	p.Mode = ModeIncognito
	p.Note = "on a call"

	out := New(p).Apply(sample())
	if out[presence.KeyIncognito] != true || out.String(presence.KeyIncognitoNote) != "on a call" {
		t.Errorf("expected an announced badge, got %v / %q", out[presence.KeyIncognito], out.String(presence.KeyIncognitoNote))
	}

	p.Announce = false
	if New(p).Apply(sample()).Has(presence.KeyIncognito) {
		t.Error("announce off must publish nothing")
	}
}

func TestExpiry(t *testing.T) {
	p := Default()
	p.Mode = ModeIncognito

	p.Until = time.Now().Add(-time.Minute).Format(time.RFC3339)
	if p.Hidden() {
		t.Error("an elapsed timer should put you back on air")
	}
	if New(p).Apply(sample()).String(presence.KeyActiveApp) != "firefox" {
		t.Error("expired incognito should publish apps again")
	}

	p.Until = time.Now().Add(time.Hour).Format(time.RFC3339)
	if !p.Hidden() {
		t.Error("a live timer should keep you hidden")
	}

	// A timestamp we cannot read must not quietly expose you.
	p.Until = "whenever"
	if !p.Hidden() {
		t.Error("an unparsable expiry should fail closed")
	}
}

func TestUnknownModeFallsBackToHiding(t *testing.T) {
	p := Default()
	p.Mode = "supersecret"

	if out := New(p).Apply(sample()); out.Has(presence.KeyActiveApp) {
		t.Error("an unknown mode must fall back to the incognito profile, not to normal")
	}
}
