package privacy

import (
	"fmt"
	"testing"
	"time"

	"statusphere-client/internal/presence"
)

func sample() presence.Snapshot {
	return presence.Snapshot{
		presence.KeyActiveApp:          "firefox",
		presence.KeyActiveWindow:       "tax return 2025 — Mozilla Firefox",
		presence.KeyActiveWorkspace:    "3",
		presence.KeyGameStatus:         "playing",
		presence.KeyGameSource:         "steam",
		presence.KeyGameAppID:          1174180,
		presence.KeyGameName:           "Red Dead Redemption 2",
		presence.KeyGameHeroURL:        "https://example.invalid/hero.jpg",
		presence.KeyGameSessionSeconds: 5040,
		presence.KeySpotifyStatus:      "playing",
		presence.KeySpotifyTrack:       "Teardrop",
		presence.KeySpotifyArtist:      "Massive Attack",
		presence.KeyMusic:              "How to file a tax return - YouTube",
		presence.KeyCPUPercent:         12.5,
		presence.KeyCustomFields:       []string{"weather"},
		"weather":                      "18°C",
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
		{"games playing keeps the fact not the title", Profile{Games: LevelPlaying}, func(s presence.Snapshot) error {
			if s.String(presence.KeyGameStatus) != "playing" || !s.Has(presence.KeyGameSessionSeconds) {
				return fmt.Errorf("the fact of playing was dropped: %v", s)
			}
			if s.Has(presence.KeyGameName) || s.Has(presence.KeyGameHeroURL) || s.Has(presence.KeyGameAppID) {
				return fmt.Errorf("the title survived: %v", s)
			}
			return nil
		}},
		{"games off drops the lot", Profile{Games: LevelOff}, func(s presence.Snapshot) error {
			for _, key := range gameKeys {
				if s.Has(key) {
					return fmt.Errorf("%s survived", key)
				}
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

// Incognito hides the game the way it hides the windows, and leaves the music.
func TestIncognitoDropsGames(t *testing.T) {
	p := Default()
	p.Mode = ModeIncognito

	out := New(p).Apply(sample())
	for _, key := range gameKeys {
		if out.Has(key) {
			t.Errorf("hidden snapshot still carries %s", key)
		}
	}
	if out.String(presence.KeySpotifyTrack) != "Teardrop" {
		t.Error("music should have stayed")
	}
	if New(Default()).Apply(sample()).String(presence.KeyGameName) != "Red Dead Redemption 2" {
		t.Error("visible mode should publish the game as before")
	}
}

// A profile written before games existed has no such key, and must resolve to the
// built-in default rather than to an empty level that drops through every switch.
func TestProfileWithoutGamesFallsBackToTheDefault(t *testing.T) {
	p := Default()
	p.Profiles = map[string]Profile{
		ModeNormal:    {Apps: LevelFull, Music: LevelFull, System: LevelOn, Custom: LevelOn},
		ModeIncognito: {Apps: LevelOff, Music: LevelFull, System: LevelOn, Custom: LevelOn},
	}

	if got := p.Active().Games; got != LevelFull {
		t.Errorf("normal games = %q, want full", got)
	}
	p.Mode = ModeIncognito
	if got := p.Active().Games; got != LevelOff {
		t.Errorf("incognito games = %q, want off", got)
	}
}

// Opting the game back in while hidden only works if its keys are classified: the
// unknown-key purge runs after every group has had its say.
func TestGamesFullSurvivesIncognito(t *testing.T) {
	out := New(policy(Profile{Apps: LevelOff, Games: LevelFull})).Apply(sample())

	if out.String(presence.KeyGameName) != "Red Dead Redemption 2" {
		t.Error("games full was asked for and the purge took it anyway")
	}
	if out.Has(presence.KeyActiveWindow) {
		t.Error("apps should still be off")
	}
}

// hide_apps names things you never want published, and a game title is one of
// them - but matching one group must not blank the other.
func TestHideAppsMatchesGamesSeparately(t *testing.T) {
	p := Default()
	p.HideApps = []string{"(?i)red dead"}

	out := New(p).Apply(sample())
	if out.Has(presence.KeyGameName) {
		t.Error("a hidden title was published")
	}
	if out.String(presence.KeyActiveApp) != "firefox" {
		t.Error("hiding the game took the window with it")
	}

	p.HideApps = []string{"(?i)firefox"}
	out = New(p).Apply(sample())
	if out.Has(presence.KeyActiveApp) {
		t.Error("a hidden app was published")
	}
	if out.String(presence.KeyGameName) != "Red Dead Redemption 2" {
		t.Error("hiding the window took the game with it")
	}
}
