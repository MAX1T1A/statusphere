package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestOverlayPlacesBoxAndKeepsSurroundings(t *testing.T) {
	base := strings.Join([]string{
		"aaaaaaaaaa",
		"bbbbbbbbbb",
		"cccccccccc",
		"dddddddddd",
	}, "\n")

	got := ansi.Strip(overlay(base, "XX\nYY", 1, 4))
	want := strings.Join([]string{
		"aaaaaaaaaa",
		"bbbbXXbbbb",
		"ccccYYcccc",
		"dddddddddd",
	}, "\n")
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestOverlayKeepsEveryLineWidth(t *testing.T) {
	base := strings.Repeat("0123456789\n", 6)
	out := overlay(strings.TrimRight(base, "\n"), "╭──╮\n│hi│\n╰──╯", 2, 3)
	for i, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w != 10 {
			t.Fatalf("line %d width = %d, want 10: %q", i, w, ansi.Strip(l))
		}
	}
}

func TestOverlayClampsInsteadOfOverflowing(t *testing.T) {
	base := strings.Join([]string{"aa", "bb", "cc"}, "\n")
	out := overlay(base, "11\n22", 5, 0)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("overlay must not grow the frame: %d lines", len(lines))
	}
	if ansi.Strip(lines[1]) != "11" || ansi.Strip(lines[2]) != "22" {
		t.Fatalf("box should be pushed up to fit: %v", lines)
	}
}

func TestOverlayPadsShortLines(t *testing.T) {
	out := ansi.Strip(overlay("ab", "ZZ", 0, 5))
	if out != "ab   ZZ" {
		t.Fatalf("got %q", out)
	}
}

func TestOverlayDoesNotCorruptStyledBase(t *testing.T) {
	style := lipgloss.NewStyle().Foreground(cAccent)
	base := style.Render("0123456789") + "\n" + style.Render("abcdefghij")

	out := overlay(base, "##", 0, 4)
	plain := ansi.Strip(out)
	if plain != "0123##6789\nabcdefghij" {
		t.Fatalf("styled base corrupted: %q", plain)
	}
	for _, l := range strings.Split(out, "\n") {
		if ansi.StringWidth(l) != 10 {
			t.Fatalf("width changed on a styled line: %q", ansi.Strip(l))
		}
	}
}

func TestOverlayNoopOnEmptyBox(t *testing.T) {
	base := "keep\nme"
	if overlay(base, "", 0, 0) != base {
		t.Fatal("an empty box must leave the frame untouched")
	}
}

func TestMusicPanelIsFramedAndSeparated(t *testing.T) {
	m := musicModel(t)
	m.expand.toggle("acc-bob", "album")

	card := renderCard(m.groups[m.selected], m.blocks, m.custom, true, 70, false,
		m.musicDetail(m.groups[m.selected], 70))
	plain := ansi.Strip(card)
	if !strings.Contains(plain, "╭") || strings.Count(plain, "╭") < 2 {
		t.Fatalf("the expanded music block should have its own frame:\n%s", plain)
	}
	if !strings.Contains(plain, "Music Has the Right") {
		t.Fatalf("album missing:\n%s", plain)
	}
}

func TestPopupFloatsOverTheFrameWithoutResizingIt(t *testing.T) {
	m := musicModel(t)
	plainBefore := ansi.Strip(m.View())

	m.mode = modeMenu
	m.openSection = "music"
	withPopup := ansi.Strip(m.View())

	if len(strings.Split(plainBefore, "\n")) != len(strings.Split(withPopup, "\n")) {
		t.Fatal("the popup must float, not push the layout down")
	}
	if !strings.Contains(withPopup, "Album art") {
		t.Fatal("popup content missing")
	}
	last := strings.Split(withPopup, "\n")
	if !strings.Contains(last[len(last)-1], "esc") {
		t.Fatalf("the footer must stay visible: %q", last[len(last)-1])
	}
}
