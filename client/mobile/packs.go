package mobile

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"statusphere-client/internal/cardlayout"
	"statusphere-client/internal/config"
	"statusphere-client/internal/layout"
	"statusphere-client/internal/presence"
)

const (
	RowSurface    = "row"
	DetailSurface = "detail"
)

// DefaultPack clears a surface: the row goes back to the header-only card
// and the detail grid to the one cardlayout builds from the phone's fields.
const DefaultPack = ""

const avatarShapeKey = "avatarShape"

type packTile struct {
	Type      string `json:"type"`
	Field     string `json:"field,omitempty"`
	Form      string `json:"form"`
	Size      string `json:"size"`
	Color     string `json:"color"`
	OnMissing string `json:"onMissing"`
}

type pack struct {
	id          string
	tiles       []packTile
	avatarShape string
}

func musicTile(form, size string) packTile {
	return packTile{Type: "music", Form: form, Size: size, Color: "primaryContainer", OnMissing: "hide"}
}

func videoTile(size string) packTile {
	return packTile{Type: "video", Form: "player", Size: size, Color: "errorContainer", OnMissing: "hide"}
}

func alarmTile(size string) packTile {
	return packTile{Type: "alarm", Form: "clock", Size: size, Color: "tertiaryContainer", OnMissing: "hide"}
}

func meetingTile(size string) packTile {
	return packTile{Type: "meeting", Form: "banner", Size: size, Color: "errorContainer", OnMissing: "hide"}
}

func appTile(form, size string) packTile {
	return packTile{Type: "scalar", Field: "active_app", Form: form, Size: size, Color: "secondaryContainer", OnMissing: "hide"}
}

func batteryTile(form, size string) packTile {
	return packTile{Type: "scalar", Field: "battery", Form: form, Size: size, Color: "tertiaryContainer", OnMissing: "dim"}
}

var packsBySurface = map[string][]pack{
	RowSurface: {
		{id: "cover", tiles: []packTile{musicTile("cover", "4x1"), videoTile("4x1")}},
		{id: "vinyl", tiles: []packTile{musicTile("vinyl", "1x1"), appTile("text", "2x1"), batteryTile("ring", "1x1"), videoTile("4x1")}, avatarShape: "SineCookie"},
		{id: "minimal", tiles: []packTile{}},
	},
	DetailSurface: {
		{id: "music", tiles: []packTile{musicTile("cover", "2x2"), appTile("text", "2x1"), batteryTile("bar", "2x1"), videoTile("4x1"), alarmTile("1x1"), meetingTile("2x1")}},
		{id: "dashboard", tiles: []packTile{musicTile("vinyl", "2x2"), batteryTile("ring", "2x2"), appTile("big", "4x1"), videoTile("4x1")}},
		{id: "compact", tiles: []packTile{musicTile("wave", "4x1"), videoTile("4x1"), appTile("text", "2x1"), batteryTile("bar", "2x1"), alarmTile("1x1"), meetingTile("2x1")}},
	},
}

func packFor(surface, id string) (*pack, error) {
	packs, ok := packsBySurface[surface]
	if !ok {
		return nil, fmt.Errorf("unknown surface %q", surface)
	}
	if id == DefaultPack {
		return nil, nil
	}
	i := slices.IndexFunc(packs, func(p pack) bool { return p.id == id })
	if i < 0 {
		return nil, fmt.Errorf("unknown %s pack %q", surface, id)
	}
	return &packs[i], nil
}

// Packs lists the pack ids of a surface as a JSON array, in display order.
func Packs(surface string) (string, error) {
	packs, ok := packsBySurface[surface]
	if !ok {
		return "", fmt.Errorf("unknown surface %q", surface)
	}
	ids := make([]string, 0, len(packs))
	for _, p := range packs {
		ids = append(ids, p.id)
	}
	data, err := json.Marshal(ids)
	return string(data), err
}

// ActivePack names the pack the saved layout matches on a surface, or
// DefaultPack when the surface is unset or no pack matches it.
func (s *Session) ActivePack(surface string) string {
	saved, set := readLayout()[surface].([]any)
	if !set {
		return DefaultPack
	}
	for _, p := range packsBySurface[surface] {
		if slices.Equal(p.tiles, tilesOf(saved)) {
			return p.id
		}
	}
	return DefaultPack
}

// SetPack saves the pack as the phone's layout.json and republishes at once,
// so friends see the new card without waiting for a heartbeat.
func (s *Session) SetPack(surface, id string) error {
	p, err := packFor(surface, id)
	if err != nil {
		return err
	}
	next := withPack(readLayout(), surface, p)
	next["updated_at"] = time.Now().Unix()
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := config.Write(layout.FileName, data, 0o600); err != nil {
		return err
	}

	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	s.layout = &layout.Store{}
	s.offerLocked(s.currentHeartbeat())
	return nil
}

// PreviewCard returns the phone's own card, as cardlayout would place it,
// with the pack applied to the surface on top of the saved layout.
func (s *Session) PreviewCard(surface, id string) (string, error) {
	p, err := packFor(surface, id)
	if err != nil {
		return "", err
	}
	s.publishMu.Lock()
	own := presence.New()
	if s.current != nil {
		own = withoutPackage(s.current.Clone())
	}
	s.publishMu.Unlock()

	own.Set(presence.KeyAccountID, s.cfg.AccountID)
	own.Set(presence.KeyLayout, withPack(readLayout(), surface, p.shownWithoutData()))
	data, err := json.Marshal(cardlayout.Cards([]presence.Snapshot{own}, nil)[0])
	return string(data), err
}

func (p *pack) shownWithoutData() *pack {
	if p == nil {
		return nil
	}
	shown := *p
	shown.tiles = slices.Clone(p.tiles)
	for i := range shown.tiles {
		shown.tiles[i].OnMissing = "dim"
	}
	return &shown
}

func readLayout() map[string]any {
	data, err := config.Read(layout.FileName)
	if err != nil {
		return map[string]any{}
	}
	var saved map[string]any
	if json.Unmarshal(data, &saved) != nil || saved == nil {
		return map[string]any{}
	}
	return saved
}

func withPack(saved map[string]any, surface string, p *pack) map[string]any {
	next := make(map[string]any, len(saved)+2)
	for k, v := range saved {
		next[k] = v
	}
	if p == nil {
		delete(next, surface)
	} else {
		next[surface] = asJSONTiles(p.tiles)
	}
	if surface == RowSurface {
		if p == nil || p.avatarShape == "" {
			delete(next, avatarShapeKey)
		} else {
			next[avatarShapeKey] = p.avatarShape
		}
	}
	return next
}

// asJSONTiles hands cardlayout the []any of maps it reads from a decoded
// layout.json, the only shape its sanitize accepts.
func asJSONTiles(tiles []packTile) []any {
	data, _ := json.Marshal(tiles)
	var out []any
	_ = json.Unmarshal(data, &out)
	return out
}

func tilesOf(raw []any) []packTile {
	data, _ := json.Marshal(raw)
	var out []packTile
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	return out
}
