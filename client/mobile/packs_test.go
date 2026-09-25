package mobile

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"statusphere-client/internal/cardlayout"
	"statusphere-client/internal/presence"
)

const fullPhone = `{"music":{"track":"Roygbiv","artist":"Boards of Canada","status":"Playing"},"video":{"title":"Rust in 100 Seconds","channel":"Fireship","status":"Playing"},"app":{"label":"Telegram","package":"org.telegram.messenger"},"battery":{"percent":80,"charging":false},"alarm_at":1790003600,"meeting_until":1790007200}`

func layoutOf(t *testing.T, frame map[string]any) map[string]any {
	t.Helper()
	l, ok := frame[presence.KeyLayout].(map[string]any)
	if !ok {
		t.Fatalf("frame should carry a layout, got %v", frame)
	}
	return l
}

func TestSetPackReachesTheRoomAndSurvivesRestart(t *testing.T) {
	srv := newFakeServer(t)
	dir := baseDir(t, srv.URL)
	s, conn, _ := startSession(t, srv, dir)
	if err := s.Publish(fullPhone); err != nil {
		t.Fatal(err)
	}
	if _, ok := conn.next(t)[presence.KeyLayout]; ok {
		t.Fatal("a phone with no pack chosen should not send a layout")
	}

	if err := s.SetPack(RowSurface, "vinyl"); err != nil {
		t.Fatal(err)
	}
	l := layoutOf(t, conn.next(t))
	if row, _ := l[RowSurface].([]any); len(row) != 4 || l[avatarShapeKey] != "SineCookie" {
		t.Fatalf("the vinyl row should arrive with its four tiles and avatar shape, got %v", l)
	}
	if _, ok := l["updated_at"].(float64); !ok {
		t.Fatalf("a layout needs updated_at to win over older devices, got %v", l)
	}

	if err := s.SetPack(DetailSurface, "music"); err != nil {
		t.Fatal(err)
	}
	l = layoutOf(t, conn.next(t))
	if _, ok := l[RowSurface]; !ok {
		t.Fatalf("choosing a detail pack should keep the row pack, got %v", l)
	}

	if err := s.SetPack(RowSurface, DefaultPack); err != nil {
		t.Fatal(err)
	}
	l = layoutOf(t, conn.next(t))
	if _, ok := l[RowSurface]; ok {
		t.Fatalf("the default row drops the surface, got %v", l)
	}
	if _, ok := l[avatarShapeKey]; ok {
		t.Fatalf("the default row drops the pack's avatar shape, got %v", l)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.ActivePack(DetailSurface); got != "music" {
		t.Fatalf("detail pack after restart = %q, want music", got)
	}
	if got := reopened.ActivePack(RowSurface); got != DefaultPack {
		t.Fatalf("row pack after restart = %q, want the default", got)
	}
}

func TestEveryPackPlacesAllItsTilesOnAPhoneWithData(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))
	if err := s.Publish(fullPhone); err != nil {
		t.Fatal(err)
	}
	conn.next(t)

	for surface, packs := range packsBySurface {
		for _, p := range packs {
			raw, err := s.PreviewCard(surface, p.id)
			if err != nil {
				t.Fatal(err)
			}
			var card cardlayout.Card
			if err := json.Unmarshal([]byte(raw), &card); err != nil {
				t.Fatal(err)
			}
			placed := card.Detail
			if surface == RowSurface {
				placed = card.Row
			}
			if len(placed) != len(p.tiles) {
				t.Errorf("%s/%s placed %d of %d tiles: %s", surface, p.id, len(placed), len(p.tiles), raw)
			}
		}
	}
	conn.none(t, 300*time.Millisecond)
	if got := s.ActivePack(DetailSurface); got != DefaultPack {
		t.Fatalf("a preview must not save the pack, active detail pack is %q", got)
	}
}

func TestPreviewShowsEveryTileDimmedBeforeThePhoneHasData(t *testing.T) {
	srv := newFakeServer(t)
	s, _, _ := startSession(t, srv, baseDir(t, srv.URL))

	raw, err := s.PreviewCard(RowSurface, "cover")
	if err != nil {
		t.Fatal(err)
	}
	var card cardlayout.Card
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		t.Fatal(err)
	}
	if len(card.Row) != 2 || !card.Row[0].Dimmed || !card.Row[1].Dimmed {
		t.Fatalf("the cover pack should preview as dimmed music and video tiles, got %s", raw)
	}
}

func TestSetPackRejectsUnknownPack(t *testing.T) {
	srv := newFakeServer(t)
	s, _, _ := startSession(t, srv, baseDir(t, srv.URL))
	if err := s.SetPack(DetailSurface, "vinyl"); err == nil {
		t.Fatal("a row pack is not a detail pack")
	}
	if err := s.SetPack("sidebar", DefaultPack); err == nil {
		t.Fatal("expected an error for an unknown surface")
	}
}

func TestCustomTilesReachTheRoomAndReadBack(t *testing.T) {
	srv := newFakeServer(t)
	dir := baseDir(t, srv.URL)
	s, conn, _ := startSession(t, srv, dir)
	if err := s.Publish(fullPhone); err != nil {
		t.Fatal(err)
	}
	conn.next(t)

	const chosen = `[{"kind":"alarm","form":"clock","size":"1x1"},{"kind":"battery","form":"ring","size":"2x2"}]`
	raw, err := s.PreviewCustom(DetailSurface, chosen)
	if err != nil {
		t.Fatal(err)
	}
	var card cardlayout.Card
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		t.Fatal(err)
	}
	if len(card.Detail) != 2 || card.Detail[0].Type != cardlayout.Alarm {
		t.Fatalf("the preview should place the alarm and the battery, got %s", raw)
	}
	conn.none(t, 300*time.Millisecond)

	if err := s.SetCustom(DetailSurface, chosen); err != nil {
		t.Fatal(err)
	}
	if detail, _ := layoutOf(t, conn.next(t))[DetailSurface].([]any); len(detail) != 2 {
		t.Fatalf("the room should get both chosen tiles, got %v", detail)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.ActivePack(DetailSurface); got != CustomPack {
		t.Fatalf("active detail pack = %q, want custom", got)
	}
	back, err := reopened.CustomTiles(DetailSurface)
	if err != nil {
		t.Fatal(err)
	}
	if back != chosen {
		t.Fatalf("custom tiles read back as %s, want %s", back, chosen)
	}
}

func TestCustomTilesOfAPackAreThatPack(t *testing.T) {
	srv := newFakeServer(t)
	s, _, _ := startSession(t, srv, baseDir(t, srv.URL))
	if err := s.SetPack(RowSurface, "cover"); err != nil {
		t.Fatal(err)
	}
	tiles, err := s.CustomTiles(RowSurface)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCustom(RowSurface, tiles); err != nil {
		t.Fatal(err)
	}
	if got := s.ActivePack(RowSurface); got != "cover" {
		t.Fatalf("editing from a pack without changes keeps the pack, got %q from %s", got, tiles)
	}
}

func TestSetCustomRejectsUnknownTiles(t *testing.T) {
	srv := newFakeServer(t)
	s, _, _ := startSession(t, srv, baseDir(t, srv.URL))
	for _, bad := range []string{
		`[{"kind":"weather","form":"text","size":"1x1"}]`,
		`[{"kind":"alarm","form":"ring","size":"1x1"}]`,
		`[{"kind":"music","form":"cover","size":"3x3"}]`,
	} {
		if err := s.SetCustom(DetailSurface, bad); err == nil {
			t.Errorf("expected %s to be rejected", bad)
		}
	}
}

func TestEmptyCustomDetailIsTheDefault(t *testing.T) {
	srv := newFakeServer(t)
	s, _, _ := startSession(t, srv, baseDir(t, srv.URL))
	if err := s.SetPack(DetailSurface, "music"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCustom(DetailSurface, `[]`); err != nil {
		t.Fatal(err)
	}
	if got := s.ActivePack(DetailSurface); got != DefaultPack {
		t.Fatalf("details with nothing picked fall back to the default grid, active pack is %q", got)
	}
}

func TestCustomFitMarksTilesTheRowHasNoRoomFor(t *testing.T) {
	raw, err := CustomFit(RowSurface, `[{"kind":"music","form":"cover","size":"2x2"},{"kind":"battery","form":"bar","size":"4x1"}]`)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Tiles []struct {
			Placed bool     `json:"placed"`
			Sizes  []string `json:"sizes"`
		} `json:"tiles"`
		Room []string `json:"room"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	beside := []string{"1x1", "2x1", "2x2"}
	if len(got.Tiles) != 2 || !got.Tiles[0].Placed || got.Tiles[1].Placed ||
		!slices.Equal(got.Tiles[0].Sizes, tileSizes) || !slices.Equal(got.Tiles[1].Sizes, beside) || !slices.Equal(got.Room, beside) {
		t.Fatalf("a full-width battery has no row left beside a 2x2 cover, got %s", raw)
	}
}
