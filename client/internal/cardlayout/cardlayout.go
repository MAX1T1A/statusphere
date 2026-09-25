// Package cardlayout places a friend card's tiles on its grid. It is a port of
// CardLayouts.js and the card functions of Statusphere.qml and CardGrid.qml in
// ii-widget-statusphere; keep the two in agreement.
package cardlayout

import (
	"cmp"
	"math"
	"regexp"
	"slices"
	"unicode/utf16"

	"statusphere-client/internal/presence"
)

type Card struct {
	AccountID string `json:"account_id"`
	// Row is nil unless the owner's layout claims the row surface; an empty
	// Row is the owner asking for a header only.
	Row    []Tile `json:"row"`
	Detail []Tile `json:"detail"`
}

type Tile struct {
	Col    int      `json:"col"`
	Row    int      `json:"row"`
	Cols   int      `json:"cols"`
	Rows   int      `json:"rows"`
	Type   TileType `json:"type"`
	Form   string   `json:"form,omitempty"`
	Color  string   `json:"color,omitempty"`
	Dimmed bool     `json:"dimmed,omitempty"`

	Field   string   `json:"field,omitempty"`
	Label   string   `json:"label,omitempty"`
	Value   string   `json:"value,omitempty"`
	Note    string   `json:"note,omitempty"`
	Icon    string   `json:"icon,omitempty"`
	Percent *float64 `json:"percent,omitempty"`

	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type TileType string

const (
	Scalar  TileType = "scalar"
	Music   TileType = "music"
	Game    TileType = "game"
	Video   TileType = "video"
	Alarm   TileType = "alarm"
	Meeting TileType = "meeting"
	Photo   TileType = "photo"
	Picture TileType = "picture"
)

type formSet struct {
	fallback string
	names    []string
}

var formsByType = map[TileType]formSet{
	Scalar:  {"text", []string{"ring", "dial", "bar", "number", "text", "big", "clock", "weather", "weatherLive", "moon", "sun"}},
	Music:   {"cover", []string{"cover", "vinyl", "wave"}},
	Game:    {"banner", []string{"banner", "timer"}},
	Video:   {"player", []string{"player"}},
	Alarm:   {"clock", []string{"clock"}},
	Meeting: {"banner", []string{"banner"}},
	Photo:   {},
	Picture: {},
}

const (
	rowKey    = "row"
	detailKey = "detail"
)

const (
	columns    = 4
	rowRows    = 2
	detailRows = 4

	shortValueLength = 8
	maxHeroes        = 3
)

const (
	hideWhenMissing = "hide"
	dimWhenMissing  = "dim"
)

type span struct{ cols, rows int }

var spansBySize = map[string]span{
	"1x1": {1, 1},
	"2x1": {2, 1},
	"2x2": {2, 2},
	"4x1": {4, 1},
}

func spanOf(size string) span {
	if s, ok := spansBySize[size]; ok {
		return s
	}
	return spansBySize["1x1"]
}

var colorRoles = []string{"primary", "secondary", "tertiary", "error", "primaryContainer", "secondaryContainer", "tertiaryContainer", "errorContainer"}

var (
	gaugeColors     = []string{"primaryContainer", "tertiaryContainer"}
	wideTextFields  = []string{presence.KeyActiveWindow}
	heroesFirst     = []string{"2x2", "2x1", "1x1", "4x1"}
	pictureURLShape = regexp.MustCompile(`^https://\S+$`)
)

type spec struct {
	kind      TileType
	form      string
	size      string
	field     string
	device    string
	color     string
	onMissing string
	url       string
}

func Cards(members []presence.Snapshot, livePhotoByAccount map[string]string) []Card {
	accounts := accountsOf(members)
	cards := make([]Card, 0, len(accounts))
	for _, a := range accounts {
		a.photoURL = livePhotoByAccount[a.id]
		cards = append(cards, a.card())
	}
	return cards
}

type account struct {
	id       string
	devices  []presence.Snapshot
	photoURL string
}

const staleGapSeconds = 45

func accountsOf(members []presence.Snapshot) []*account {
	var order []*account
	byID := map[string]*account{}
	for _, m := range members {
		id := cmp.Or(m.String(presence.KeyAccountID), m.DeviceID())
		if id == "" {
			continue
		}
		a := byID[id]
		if a == nil {
			a = &account{id: id}
			byID[id] = a
			order = append(order, a)
		}
		if offline, _ := m[presence.KeyOffline].(bool); offline {
			continue
		}
		a.devices = append(a.devices, m)
	}
	for _, a := range order {
		sortDevices(a.devices)
	}
	return order
}

func sortDevices(devices []presence.Snapshot) {
	newest := 0.0
	for _, d := range devices {
		seen, _ := d.Float(presence.KeyLastSeen)
		newest = max(newest, seen)
	}
	behind := func(d presence.Snapshot) int {
		if seen, _ := d.Float(presence.KeyLastSeen); newest-seen > staleGapSeconds {
			return 1
		}
		return 0
	}
	slices.SortStableFunc(devices, func(a, b presence.Snapshot) int {
		return cmp.Or(deviceRank(a)-deviceRank(b), behind(a)-behind(b), cmp.Compare(a.DeviceID(), b.DeviceID()))
	})
}

func deviceRank(d presence.Snapshot) int {
	switch {
	case d.String(presence.KeyGameStatus) == presence.GameStatusPlaying:
		return 0
	case d.String(presence.KeySpotifyStatus) == "playing":
		return 1
	case d.String(presence.KeySpotifyStatus) != "":
		return 2
	default:
		return 3
	}
}

func (a *account) primary() presence.Snapshot {
	if len(a.devices) == 0 {
		return nil
	}
	return a.devices[0]
}

func (a *account) hidden() bool {
	incognito, _ := a.primary()[presence.KeyIncognito].(bool)
	return incognito
}

func (a *account) card() Card {
	c := Card{AccountID: a.id, Detail: []Tile{}}
	if a.hidden() {
		return c
	}
	layout := a.layout()
	if row, owned := layout[rowKey].([]any); owned {
		c.Row = a.place(a.expandWildcards(row), rowRows)
	}
	detail, _ := layout[detailKey].([]any)
	tiles := a.expandWildcards(detail)
	if len(tiles) == 0 {
		tiles = standardDetailFor(detailFieldsFor(a.primary()))
	}
	c.Detail = a.place(tiles, detailRows)
	return c
}

func (a *account) layout() map[string]any {
	var best map[string]any
	for _, d := range a.devices {
		l, ok := d[presence.KeyLayout].(map[string]any)
		if ok && (best == nil || updatedAt(l) > updatedAt(best)) {
			best = l
		}
	}
	return best
}

func updatedAt(layout map[string]any) float64 {
	switch v := layout["updated_at"].(type) {
	case nil:
		return 0
	case float64:
		return v
	default:
		return math.NaN()
	}
}

func (a *account) expandWildcards(raw []any) []spec {
	var clean []spec
	named := map[string]bool{}
	for _, r := range raw {
		if t, ok := sanitize(r); ok {
			clean = append(clean, t)
			if t.field != wildcardField {
				named[t.field] = true
			}
		}
	}
	var out []spec
	for _, t := range clean {
		if t.field != wildcardField {
			out = append(out, t)
			continue
		}
		for _, f := range detailFieldsFor(a.primary()) {
			if !named[f.key] {
				t.field = f.key
				out = append(out, t)
			}
		}
	}
	return out
}

const wildcardField = "*"

func sanitize(raw any) (spec, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return spec{}, false
	}
	kind, _ := m["type"].(string)
	forms, known := formsByType[TileType(kind)]
	if !known {
		return spec{}, false
	}
	t := spec{kind: TileType(kind)}
	t.size, ok = stringIfSet(m, "size")
	if !ok || (t.size != "" && spansBySize[t.size] == (span{})) {
		return spec{}, false
	}
	t.field, _ = m["field"].(string)
	if t.kind == Scalar && t.field == "" {
		return spec{}, false
	}
	t.form, ok = stringIfSet(m, "form")
	if len(forms.names) > 0 && (!ok || (t.form != "" && !slices.Contains(forms.names, t.form))) {
		return spec{}, false
	}
	t.device, _ = m["device"].(string)
	t.color, _ = m["color"].(string)
	t.onMissing, _ = m["onMissing"].(string)
	if url, _ := m["url"].(string); pictureURLShape.MatchString(url) {
		t.url = url
	}
	return t, true
}

func stringIfSet(m map[string]any, key string) (string, bool) {
	v, set := m[key]
	if !set {
		return "", true
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func (t spec) resolvedForm() string {
	forms := formsByType[t.kind]
	if slices.Contains(forms.names, t.form) {
		return t.form
	}
	return forms.fallback
}

func standardDetailFor(fields []field) []spec {
	tiles := make([]spec, 0, len(fields))
	gauges := 0
	for _, f := range fields {
		tiles = append(tiles, standardTileFor(f, &gauges))
	}
	var best []spec
	var bestScore [3]int
	for _, widenText := range []bool{false, true} {
		for heroes := 0; heroes <= maxHeroes; heroes++ {
			candidate := grown(tiles, heroes, widenText)
			placed := pack(candidate, detailRows)
			score := [3]int{emptyCells(placed), len(candidate) - len(placed), heroes + boolInt(widenText)}
			if best == nil || slices.Compare(score[:], bestScore[:]) < 0 {
				best, bestScore = candidate, score
			}
		}
	}
	return best
}

func standardTileFor(f field, gauges *int) spec {
	t := spec{kind: Scalar, field: f.key, color: "secondaryContainer", onMissing: hideWhenMissing}
	switch {
	case f.percent != nil:
		t.form, t.size = "ring", "1x1"
		t.color = gaugeColors[*gauges%len(gaugeColors)]
		*gauges++
	case slices.Contains(wideTextFields, f.key) || len(utf16.Encode([]rune(f.value))) > shortValueLength:
		t.form, t.size = "text", "2x1"
	default:
		t.form, t.size = "number", "1x1"
	}
	return t
}

func grown(tiles []spec, heroes int, widenText bool) []spec {
	var heroIdx []int
	for _, form := range []string{"ring", "number"} {
		for i, t := range tiles {
			if t.form == form {
				heroIdx = append(heroIdx, i)
			}
		}
	}
	heroIdx = heroIdx[:min(heroes, len(heroIdx))]
	textIdx := -1
	if widenText {
		textIdx = slices.IndexFunc(tiles, func(t spec) bool { return t.size == "2x1" })
	}
	out := slices.Clone(tiles)
	for i := range out {
		switch {
		case slices.Contains(heroIdx, i):
			out[i].size = "2x2"
		case i == textIdx:
			out[i].size = "4x1"
		}
	}
	sorted := make([]spec, 0, len(out))
	for _, size := range heroesFirst {
		for _, t := range out {
			if t.size == size {
				sorted = append(sorted, t)
			}
		}
	}
	return sorted
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type placement struct {
	index      int
	col, row   int
	cols, rows int
}

func pack(tiles []spec, maxRows int) []placement {
	var occupied [][columns]bool
	free := func(col, row int, s span) bool {
		for r := row; r < row+s.rows && r < len(occupied); r++ {
			for c := col; c < col+s.cols; c++ {
				if occupied[r][c] {
					return false
				}
			}
		}
		return true
	}
	var placed []placement
	for i, t := range tiles {
		s := spanOf(t.size)
		spot, found := placement{}, false
		for row := 0; row+s.rows <= maxRows && !found; row++ {
			for col := 0; col+s.cols <= columns && !found; col++ {
				if free(col, row, s) {
					spot, found = placement{index: i, col: col, row: row, cols: s.cols, rows: s.rows}, true
				}
			}
		}
		if !found {
			continue
		}
		for len(occupied) < spot.row+spot.rows {
			occupied = append(occupied, [columns]bool{})
		}
		for r := spot.row; r < spot.row+spot.rows; r++ {
			for c := spot.col; c < spot.col+spot.cols; c++ {
				occupied[r][c] = true
			}
		}
		placed = append(placed, spot)
	}
	return placed
}

func emptyCells(placed []placement) int {
	rowsUsed, filled := 0, 0
	for _, p := range placed {
		rowsUsed = max(rowsUsed, p.row+p.rows)
		filled += p.cols * p.rows
	}
	return rowsUsed*columns - filled
}

func (a *account) place(tiles []spec, maxRows int) []Tile {
	var shown []spec
	for _, t := range tiles {
		if t.onMissing != hideWhenMissing || a.hasData(t) {
			shown = append(shown, t)
		}
	}
	placed := pack(shown, maxRows)
	out := make([]Tile, 0, len(placed))
	for _, p := range placed {
		out = append(out, a.tile(shown[p.index], p))
	}
	return out
}

func (a *account) deviceFor(t spec) presence.Snapshot {
	if t.device == "" {
		return a.primary()
	}
	for _, d := range a.devices {
		if d.DeviceID() == t.device {
			return d
		}
	}
	return nil
}

func (a *account) hasData(t spec) bool {
	switch t.kind {
	case Scalar:
		return fieldFor(a.deviceFor(t), t.field) != nil
	case Music:
		return len(a.musicDevices()) > 0
	case Game:
		return len(a.gameDevices()) > 0
	case Video:
		return len(a.videoDevices()) > 0
	case Alarm:
		return len(a.alarmDevices()) > 0
	case Meeting:
		return len(a.meetingDevices()) > 0
	case Photo:
		return a.photoURL != ""
	case Picture:
		return t.url != ""
	}
	return false
}

func (a *account) tile(t spec, p placement) Tile {
	out := Tile{
		Col: p.col, Row: p.row, Cols: p.cols, Rows: p.rows,
		Type:   t.kind,
		Form:   t.resolvedForm(),
		Dimmed: t.onMissing == dimWhenMissing && !a.hasData(t),
	}
	if slices.Contains(colorRoles, t.color) {
		out.Color = t.color
	}
	switch t.kind {
	case Scalar:
		out.Field = t.field
		out.Label = labelForKey(t.field)
		if f := fieldFor(a.deviceFor(t), t.field); f != nil {
			out.Label, out.Value, out.Note, out.Icon, out.Percent = f.label, f.value, f.note, f.icon, f.percent
		}
	case Music:
		if devices := a.musicDevices(); len(devices) > 0 {
			d := devices[0]
			out.Title = cmp.Or(d.String(presence.KeySpotifyTrack), d.String(presence.KeySpotifyDisplay))
			out.Subtitle = d.String(presence.KeySpotifyArtist)
			out.ImageURL = d.String(presence.KeySpotifyArtURL)
		}
	case Game:
		if devices := a.gameDevices(); len(devices) > 0 {
			d := devices[0]
			out.Title = cmp.Or(d.String(presence.KeyGameDisplay), d.String(presence.KeyGameName))
			out.ImageURL = cmp.Or(d.String(presence.KeyGameHeroURL), d.String(presence.KeyGameHeaderURL))
		}
	case Video:
		if devices := a.videoDevices(); len(devices) > 0 {
			d := devices[0]
			out.Title = d.String(presence.KeyVideoTitle)
			out.Subtitle = d.String(presence.KeyVideoChannel)
			position, _ := d.Float(presence.KeyVideoPosition)
			if length, _ := d.Float(presence.KeyVideoLength); length > 0 {
				progress := position / length * 100
				out.Percent = &progress
			}
		}
	case Alarm:
		if devices := a.alarmDevices(); len(devices) > 0 {
			if at, ok := devices[0].Float(presence.KeyAlarmAt); ok {
				out.Value = jsNumber(at)
			}
		}
	case Meeting:
		if devices := a.meetingDevices(); len(devices) > 0 {
			if until, ok := devices[0].Float(presence.KeyMeetingUntil); ok {
				out.Value = jsNumber(until)
			}
		}
	case Photo:
		out.ImageURL = a.photoURL
	case Picture:
		out.ImageURL = t.url
	}
	return out
}

func (a *account) musicDevices() []presence.Snapshot {
	return distinct(a.devices, func(d presence.Snapshot) (string, bool) {
		trackKey := cmp.Or(d.String(presence.KeySpotifyURI), d.String(presence.KeySpotifyDisplay),
			d.String(presence.KeySpotifyTrack)+"/"+d.String(presence.KeySpotifyArtist))
		return trackKey, d.String(presence.KeySpotifyStatus) != ""
	})
}

func (a *account) gameDevices() []presence.Snapshot {
	return distinct(a.devices, func(d presence.Snapshot) (string, bool) {
		playing := d.String(presence.KeyGameStatus) != "" && d.String(presence.KeyGameName) != ""
		if appID := d[presence.KeyGameAppID]; truthy(appID) {
			return jsString(appID), playing
		}
		return d.String(presence.KeyGameName), playing
	})
}

func (a *account) videoDevices() []presence.Snapshot {
	return distinct(a.devices, func(d presence.Snapshot) (string, bool) {
		title := d.String(presence.KeyVideoTitle)
		return title + "/" + d.String(presence.KeyVideoChannel), d.String(presence.KeyVideoStatus) != "" && title != ""
	})
}

func (a *account) alarmDevices() []presence.Snapshot {
	return distinct(a.devices, func(d presence.Snapshot) (string, bool) {
		at, ok := d.Float(presence.KeyAlarmAt)
		return jsNumber(at), ok
	})
}

func (a *account) meetingDevices() []presence.Snapshot {
	return distinct(a.devices, func(d presence.Snapshot) (string, bool) {
		until, ok := d.Float(presence.KeyMeetingUntil)
		return jsNumber(until), ok
	})
}

func distinct(devices []presence.Snapshot, keyOf func(presence.Snapshot) (key string, wanted bool)) []presence.Snapshot {
	var out []presence.Snapshot
	seen := map[string]bool{}
	for _, d := range devices {
		key, wanted := keyOf(d)
		if !wanted || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	return out
}
