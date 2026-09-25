package mobile

import (
	"encoding/json"
	"fmt"
	"maps"
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

// CustomPack is what ActivePack reports for a surface the owner assembled
// tile by tile through SetCustom.
const CustomPack = "custom"

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

type tileKind struct {
	id    string
	forms []string
	build func(form, size string) packTile
}

var tileKinds = []tileKind{
	{"music", []string{"cover", "vinyl", "wave"}, musicTile},
	{"video", []string{"player"}, func(_, size string) packTile { return videoTile(size) }},
	{"app", []string{"text", "big"}, appTile},
	{"battery", []string{"bar", "ring", "number"}, batteryTile},
	{"alarm", []string{"clock"}, func(_, size string) packTile { return alarmTile(size) }},
	{"meeting", []string{"banner"}, func(_, size string) packTile { return meetingTile(size) }},
}

var tileSizes = []string{"1x1", "2x1", "2x2", "4x1"}

type customTile struct {
	Kind string `json:"kind"`
	Form string `json:"form"`
	Size string `json:"size"`
}

func (k tileKind) matches(t packTile) bool {
	return slices.Contains(k.forms, t.Form) && slices.Contains(tileSizes, t.Size) && k.build(t.Form, t.Size) == t
}

// TileKinds lists what SetCustom accepts, in display order, as a JSON array
// of {"kind", "forms", "sizes"}.
func TileKinds() (string, error) {
	type entry struct {
		Kind  string   `json:"kind"`
		Forms []string `json:"forms"`
		Sizes []string `json:"sizes"`
	}
	out := make([]entry, 0, len(tileKinds))
	for _, k := range tileKinds {
		out = append(out, entry{k.id, k.forms, tileSizes})
	}
	data, err := json.Marshal(out)
	return string(data), err
}

// CustomTiles returns the saved tiles of a surface as SetCustom takes them,
// skipping those no tile kind builds.
func (s *Session) CustomTiles(surface string) (string, error) {
	if _, ok := packsBySurface[surface]; !ok {
		return "", fmt.Errorf("unknown surface %q", surface)
	}
	saved, _ := readLayout()[surface].([]any)
	out := []customTile{}
	for _, t := range tilesOf(saved) {
		i := slices.IndexFunc(tileKinds, func(k tileKind) bool { return k.matches(t) })
		if i >= 0 {
			out = append(out, customTile{tileKinds[i].id, t.Form, t.Size})
		}
	}
	data, err := json.Marshal(out)
	return string(data), err
}

// CustomFit tells the editor where tiles as SetCustom takes them stand in
// the preview grid, as JSON {"tiles": [{"placed", "sizes"}], "room"}: whether
// each tile got a spot, the sizes it would get one at with the rest kept, and
// the sizes a tile appended after them would get one at.
func CustomFit(surface, tilesJSON string) (string, error) {
	type tileFit struct {
		Placed bool     `json:"placed"`
		Sizes  []string `json:"sizes"`
	}
	chosen, err := decodeCustom(surface, tilesJSON)
	if err != nil {
		return "", err
	}
	sizes := make([]string, len(chosen))
	for i, c := range chosen {
		sizes[i] = c.Size
	}
	fitsAt := func(i int) []string {
		var out []string
		for _, size := range tileSizes {
			tried := slices.Clone(sizes)
			if i == len(sizes) {
				tried = append(tried, size)
			} else {
				tried[i] = size
			}
			if cardlayout.Placed(surface, tried)[i] {
				out = append(out, size)
			}
		}
		return out
	}
	placed := cardlayout.Placed(surface, sizes)
	fit := struct {
		Tiles []tileFit `json:"tiles"`
		Room  []string  `json:"room"`
	}{Tiles: make([]tileFit, len(chosen)), Room: fitsAt(len(chosen))}
	for i := range chosen {
		fit.Tiles[i] = tileFit{placed[i], fitsAt(i)}
	}
	data, err := json.Marshal(fit)
	return string(data), err
}

func decodeCustom(surface, tilesJSON string) ([]customTile, error) {
	if _, ok := packsBySurface[surface]; !ok {
		return nil, fmt.Errorf("unknown surface %q", surface)
	}
	var chosen []customTile
	if err := json.Unmarshal([]byte(tilesJSON), &chosen); err != nil {
		return nil, err
	}
	for _, c := range chosen {
		i := slices.IndexFunc(tileKinds, func(k tileKind) bool { return k.id == c.Kind })
		if i < 0 {
			return nil, fmt.Errorf("unknown tile kind %q", c.Kind)
		}
		if !slices.Contains(tileKinds[i].forms, c.Form) || !slices.Contains(tileSizes, c.Size) {
			return nil, fmt.Errorf("tile %s cannot be %s at %s", c.Kind, c.Form, c.Size)
		}
	}
	return chosen, nil
}

func customPack(surface, tilesJSON string) (*pack, error) {
	chosen, err := decodeCustom(surface, tilesJSON)
	if err != nil {
		return nil, err
	}
	p := &pack{id: CustomPack, tiles: make([]packTile, 0, len(chosen))}
	for _, c := range chosen {
		i := slices.IndexFunc(tileKinds, func(k tileKind) bool { return k.id == c.Kind })
		p.tiles = append(p.tiles, tileKinds[i].build(c.Form, c.Size))
	}
	if surface == DetailSurface && len(p.tiles) == 0 {
		return nil, nil
	}
	return p, nil
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

// ActivePack names the pack the saved layout matches on a surface:
// DefaultPack when the surface is unset, CustomPack when no pack matches it.
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
	return CustomPack
}

// SetPack saves the pack as the phone's layout.json and republishes at once,
// so friends see the new card without waiting for a heartbeat.
func (s *Session) SetPack(surface, id string) error {
	p, err := packFor(surface, id)
	if err != nil {
		return err
	}
	return s.save(surface, p)
}

// SetCustom saves tiles, a JSON array of {"kind", "form", "size"} from
// TileKinds, as the surface's layout and republishes like SetPack.
func (s *Session) SetCustom(surface, tilesJSON string) error {
	p, err := customPack(surface, tilesJSON)
	if err != nil {
		return err
	}
	return s.save(surface, p)
}

func (s *Session) save(surface string, p *pack) error {
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
	return s.preview(surface, p, false)
}

// PreviewCustom is PreviewCard for tiles as SetCustom takes them. With
// sample set, every tile kind the phone has no data for shows made-up data.
func (s *Session) PreviewCustom(surface, tilesJSON string, sample bool) (string, error) {
	p, err := customPack(surface, tilesJSON)
	if err != nil {
		return "", err
	}
	return s.preview(surface, p, sample)
}

func (s *Session) preview(surface string, p *pack, sample bool) (string, error) {
	s.publishMu.Lock()
	own := presence.New()
	if s.current != nil {
		own = withoutPackage(s.current.Clone())
	}
	s.publishMu.Unlock()

	if sample {
		fillMissingWithSamples(own, time.Now())
	}

	own.Set(presence.KeyAccountID, s.cfg.AccountID)
	own.Set(presence.KeyLayout, withPack(readLayout(), surface, p.shownWithoutData()))
	data, err := json.Marshal(cardlayout.Cards([]presence.Snapshot{own}, nil)[0])
	return string(data), err
}

func fillMissingWithSamples(own presence.Snapshot, now time.Time) {
	for _, part := range sampleParts(now) {
		sample := part.presence()
		if !slices.ContainsFunc(slices.Collect(maps.Keys(sample)), own.Has) {
			maps.Copy(own, sample)
		}
	}
}

func sampleParts(now time.Time) []phoneSnapshot {
	alarmAt := now.Add(7 * time.Hour).Unix()
	meetingUntil := now.Add(40 * time.Minute).Unix()
	return []phoneSnapshot{
		{Music: &phoneMusic{Track: "Midnight City", Artist: "M83", Status: "Playing", PositionSeconds: 95, LengthSeconds: 243}},
		{Video: &phoneVideo{Title: "Rust in 100 Seconds", Channel: "Fireship", Status: "Playing", PositionSeconds: 40, LengthSeconds: 150}},
		{App: &phoneApp{Label: "Telegram"}},
		{Battery: &phoneBattery{Percent: 76}},
		{AlarmAt: &alarmAt},
		{MeetingUntil: &meetingUntil},
	}
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
