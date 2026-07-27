package tui

import (
	"strings"
	"testing"

	"statusphere-client/internal/presence"
)

func musicModel(t *testing.T) model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel(Options{LocalAccountID: meAccount})
	m.width, m.height = 110, 32
	m.groups = groupDevices([]presence.Snapshot{
		{presence.KeyDeviceID: "me-1", presence.KeyAccountID: meAccount, presence.KeyAccountName: "Me", presence.KeyLastSeen: int64(0)},
		{presence.KeyDeviceID: "bob-1", presence.KeyAccountID: "acc-bob", presence.KeyAccountName: "Bob", presence.KeyLastSeen: int64(0),
			presence.KeySpotifyStatus: "playing", presence.KeySpotifyArtist: "Boards of Canada",
			presence.KeySpotifyTrack: "Roygbiv", presence.KeySpotifyAlbum: "Music Has the Right",
			presence.KeySpotifyDisplay: "Boards of Canada — Roygbiv", presence.KeySpotifyURI: "spotify:track:x",
			presence.KeySpotifyPosition: int64(30), presence.KeySpotifyLength: int64(225)},
		{presence.KeyDeviceID: "ann-1", presence.KeyAccountID: "acc-ann", presence.KeyAccountName: "Ann", presence.KeyLastSeen: int64(0),
			presence.KeySpotifyStatus: "playing", presence.KeySpotifyDisplay: "Aphex Twin — Xtal",
			presence.KeySpotifyAlbum: "Selected Ambient Works"},
	}, meAccount)
	m.clampSelection()
	selectAccount(&m, "acc-bob")
	return m
}

func TestPiecesAreOffByDefault(t *testing.T) {
	m := musicModel(t)
	if got := m.musicDetail(m.groups[m.selected], 70); got != nil {
		t.Fatalf("nothing should be expanded before you pick it: %v", got)
	}
}

func TestTogglingAPieceExpandsOnlyThatPiece(t *testing.T) {
	m := musicModel(t)
	m.expand.toggle("acc-bob", "album")

	lines := strings.Join(m.musicDetail(m.groups[m.selected], 70), "\n")
	if !strings.Contains(lines, "Music Has the Right") {
		t.Fatalf("album should be expanded:\n%s", lines)
	}
	if strings.Contains(lines, "Lyrics") || strings.Contains(lines, "this week") {
		t.Fatalf("only the picked piece may render:\n%s", lines)
	}
}

func TestExpansionIsPerPerson(t *testing.T) {
	m := musicModel(t)
	m.expand.toggle("acc-bob", "album")

	var bob, ann deviceGroup
	for _, g := range m.groups {
		switch g.key {
		case "acc-bob":
			bob = g
		case "acc-ann":
			ann = g
		}
	}
	if len(m.musicDetail(bob, 70)) == 0 {
		t.Fatal("bob should have his album expanded")
	}
	if len(m.musicDetail(ann, 70)) != 0 {
		t.Fatal("ann must not inherit bob's picks")
	}
}

func TestPicksSurviveRestart(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	first := newExpandState()
	first.toggle("acc-bob", "lyrics")
	first.toggle("acc-bob", "cover")

	reloaded := newExpandState()
	if !reloaded.enabled("acc-bob", "lyrics") || !reloaded.enabled("acc-bob", "cover") {
		t.Fatal("picks should be restored on the next run")
	}
	if reloaded.enabled("acc-ann", "lyrics") {
		t.Fatal("picks must stay scoped to their account")
	}

	first.toggle("acc-bob", "lyrics")
	if again := newExpandState(); again.enabled("acc-bob", "lyrics") {
		t.Fatal("unticking should persist too")
	}
}

func TestEnterOnMusicOpensThePopupNotAModal(t *testing.T) {
	m := musicModel(t)
	m = send(m, key("enter"))
	if m.mode != modeMenu {
		t.Fatalf("enter should open the person menu, got %v", m.mode)
	}
	m.menuIndex = actionIndex(m.personMenu(), "music")
	next, _ := m.runMenu()
	m = next.(model)
	if m.mode != modeMusicPicks {
		t.Fatalf("Music should open the picker, got %v", m.mode)
	}

	view := m.View()
	if !strings.Contains(view, "Album art") || !strings.Contains(view, "[ ]") {
		t.Fatalf("popup should list checkboxes:\n%s", view)
	}
	if !strings.Contains(view, "Bob") || !strings.Contains(view, "group chat") {
		t.Fatal("the popup must sit on the main screen, not replace it")
	}
}

func TestPopupTogglesAndClosesBackToRoom(t *testing.T) {
	m := musicModel(t)
	m.mode = modeMusicPicks
	m.pickIndex = actionIndexPiece("progress")

	next, _ := m.musicPickKeys(key("enter"))
	m = next.(model)
	if !m.expand.enabled("acc-bob", "progress") {
		t.Fatal("enter should tick the highlighted piece")
	}
	next, _ = m.musicPickKeys(key("enter"))
	m = next.(model)
	if m.expand.enabled("acc-bob", "progress") {
		t.Fatal("enter again should untick it")
	}

	next, _ = m.musicPickKeys(key("esc"))
	if next.(model).mode != modeNone {
		t.Fatal("esc should return to the room")
	}
}

func TestPopupNavigationStaysInRange(t *testing.T) {
	m := musicModel(t)
	m.mode = modeMusicPicks
	m.pickIndex = 0
	next, _ := m.musicPickKeys(key("up"))
	if next.(model).pickIndex != 0 {
		t.Fatal("up at the top should stay")
	}
	m.pickIndex = len(musicPieces) - 1
	next, _ = m.musicPickKeys(key("down"))
	if next.(model).pickIndex != len(musicPieces)-1 {
		t.Fatal("down at the bottom should stay")
	}
}

func actionIndexPiece(id string) int {
	for i, p := range musicPieces {
		if p.id == id {
			return i
		}
	}
	return 0
}
