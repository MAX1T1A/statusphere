package mobile

import (
	"encoding/json"
	"testing"
	"time"

	"statusphere-client/internal/cardlayout"
	"statusphere-client/internal/presence"
)

const fullPhone = `{"music":{"track":"Roygbiv","artist":"Boards of Canada","status":"Playing"},"app":{"label":"Telegram","package":"org.telegram.messenger"},"battery":{"percent":80,"charging":false}}`

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
	if row, _ := l[RowSurface].([]any); len(row) != 3 || l[avatarShapeKey] != "SineCookie" {
		t.Fatalf("the vinyl row should arrive with its three tiles and avatar shape, got %v", l)
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
	if len(card.Row) != 1 || !card.Row[0].Dimmed {
		t.Fatalf("the cover pack should preview as one dimmed music tile, got %s", raw)
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
