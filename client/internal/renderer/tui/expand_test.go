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

func pieceRowIndex(rows []menuRow, id string) int {
	for i, r := range rows {
		if r.kind == rowCheck && r.id == id {
			return i
		}
	}
	return -1
}

func TestMusicSectionExpandsInsideTheMenu(t *testing.T) {
	m := musicModel(t)
	m = send(m, key("enter"))
	if m.mode != modeMenu {
		t.Fatalf("enter should open the menu, got %v", m.mode)
	}
	rows := m.menuRows()
	if rows[0].kind != rowSection || rows[0].id != "music" {
		t.Fatalf("Music should be a collapsible section: %+v", rows[0])
	}
	if pieceRowIndex(rows, "lyrics") != -1 {
		t.Fatal("a collapsed section must not list its checkboxes")
	}

	m.menuIndex = 0
	m = send(m, key("right"))
	rows = m.menuRows()
	if pieceRowIndex(rows, "lyrics") == -1 {
		t.Fatal("right arrow should unfold the checkboxes under the heading")
	}

	m = send(m, key("left"))
	if pieceRowIndex(m.menuRows(), "lyrics") != -1 {
		t.Fatal("left arrow should fold it back")
	}
}

func TestTogglingACheckboxRowFromTheMenu(t *testing.T) {
	m := musicModel(t)
	m.mode = modeMenu
	m.openSection = "music"
	m.menuIndex = pieceRowIndex(m.menuRows(), "progress")

	m = send(m, key("enter"))
	if !m.expand.enabled("acc-bob", "progress") {
		t.Fatal("enter on a checkbox row should tick it")
	}
	m = send(m, key("enter"))
	if m.expand.enabled("acc-bob", "progress") {
		t.Fatal("enter again should untick it")
	}
}

func TestMenuIsACenteredPopupOverTheRoom(t *testing.T) {
	m := musicModel(t)
	plain := len(strings.Split(m.View(), "\n"))

	m.mode = modeMenu
	m.openSection = "music"
	withMenu := m.View()
	if len(strings.Split(withMenu, "\n")) != plain {
		t.Fatal("the menu popup must not resize the frame")
	}
	for _, want := range []string{"Album art", "Bob", "group chat"} {
		if !strings.Contains(withMenu, want) {
			t.Fatalf("expected %q to be visible with the menu open:\n%s", want, withMenu)
		}
	}
}
