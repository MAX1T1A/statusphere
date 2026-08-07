// Package privacy decides what leaves this machine. The filter runs on the
// snapshot before it is sent, so a hidden field reaches neither the room nor
// the server's history - nothing to leak back through screen time later.
package privacy

import (
	"encoding/json"
	"regexp"
	"time"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

const FileName = "privacy.json"

const (
	ModeNormal    = "normal"
	ModeIncognito = "incognito"
)

// Level is how much of a group is published. Groups take different sets: apps
// full/app/busy/off, games full/playing/off, music full/listening/off, the rest
// on/off.
type Level string

const (
	LevelFull      Level = "full"
	LevelApp       Level = "app"
	LevelBusy      Level = "busy"
	LevelPlaying   Level = "playing"
	LevelListening Level = "listening"
	LevelOn        Level = "on"
	LevelOff       Level = "off"
)

// BusyLabel stands in for the app name at LevelBusy: the room sees you're at
// the keyboard, not what you're doing.
const BusyLabel = "Active"

type Profile struct {
	Apps   Level `json:"apps"`
	Games  Level `json:"games"`
	Music  Level `json:"music"`
	System Level `json:"system"`
	Custom Level `json:"custom"`
}

type Policy struct {
	Mode     string             `json:"mode"`
	Until    string             `json:"until,omitempty"`
	Note     string             `json:"note,omitempty"`
	Announce bool               `json:"announce"`
	Profiles map[string]Profile `json:"profiles"`
	HideApps []string           `json:"hide_apps,omitempty"`
}

// Default keeps music on while hidden: what you listen to is the sociable part,
// what you have open is the intrusive one. A game goes with the apps - it is the
// most conspicuous thing on the screen, and hiding is usually about that.
func Default() Policy {
	return Policy{
		Mode:     ModeNormal,
		Announce: true,
		Profiles: map[string]Profile{
			ModeNormal:    {Apps: LevelFull, Games: LevelFull, Music: LevelFull, System: LevelOn, Custom: LevelOn},
			ModeIncognito: {Apps: LevelOff, Games: LevelOff, Music: LevelFull, System: LevelOn, Custom: LevelOn},
		},
		HideApps: []string{"(?i)keepassxc", "(?i)1password", "(?i)bitwarden"},
	}
}

var (
	appKeys = []string{presence.KeyActiveApp, presence.KeyActiveWindow, presence.KeyActiveWorkspace}

	gameDetailKeys = []string{
		presence.KeyGameAppID, presence.KeyGameName, presence.KeyGameDisplay,
		presence.KeyGameHeaderURL, presence.KeyGameCoverURL,
		presence.KeyGameHeroURL, presence.KeyGameLogoURL,
	}
	gameKeys = append([]string{
		presence.KeyGameSource, presence.KeyGameStatus,
		presence.KeyGameStartedAt, presence.KeyGameSessionSeconds,
	}, gameDetailKeys...)

	musicDetailKeys = []string{
		presence.KeyMusic,
		presence.KeySpotifyTrack, presence.KeySpotifyArtist, presence.KeySpotifyAlbum,
		presence.KeySpotifyArtURL, presence.KeySpotifyURI, presence.KeySpotifyDisplay,
		presence.KeySpotifyPosition, presence.KeySpotifyLength,
	}
	musicKeys = append([]string{presence.KeySpotifyStatus}, musicDetailKeys...)

	systemKeys = []string{
		presence.KeyUptimeHours, presence.KeyCPUPercent, presence.KeyCPUCount, presence.KeyMemUsedMB,
		presence.KeyMemTotalMB, presence.KeyLoadAvg1m, presence.KeyPackageCount,
		presence.KeyDiskUsedPercent, presence.KeyDiskFreeGB,
		presence.KeyHealth, presence.KeyHealthNote,
	}

	// knownKeys is what the filter can classify. Anything else is a field some
	// newer collector added, and while hidden it gets dropped rather than walk
	// out unclassified.
	knownKeys = keySet(
		presence.KeyDeviceID, presence.KeyAccountID, presence.KeyAccountName, presence.KeyDeviceName,
		presence.KeyLastSeen, presence.KeyNudge, presence.KeyCustomFields,
		presence.KeyRole, presence.KeyOffline, presence.KeyIncognito, presence.KeyIncognitoNote,
		presence.KeyKind,
	)
)

func keySet(extra ...string) map[string]bool {
	set := make(map[string]bool)
	for _, group := range [][]string{appKeys, gameKeys, musicKeys, systemKeys, extra} {
		for _, k := range group {
			set[k] = true
		}
	}
	return set
}

// Hidden reports whether the incognito profile is in force right now. An
// unreadable expiry counts as still hidden - a broken timestamp must not
// quietly put you back on air.
func (p Policy) Hidden() bool {
	if p.Mode == "" || p.Mode == ModeNormal {
		return false
	}
	if p.Until == "" {
		return true
	}
	until, err := time.Parse(time.RFC3339, p.Until)
	if err != nil {
		return true
	}
	return time.Now().Before(until)
}

func (p Policy) Expires() (time.Time, bool) {
	if p.Until == "" || !p.Hidden() {
		return time.Time{}, false
	}
	until, err := time.Parse(time.RFC3339, p.Until)
	return until, err == nil
}

// Active resolves the profile to publish by. An unknown mode name falls back to
// the built-in incognito profile rather than to normal.
func (p Policy) Active() Profile {
	def := Default().Profiles
	if !p.Hidden() {
		return fill(p.Profiles[ModeNormal], def[ModeNormal])
	}
	if prof, ok := p.Profiles[p.Mode]; ok {
		return fill(prof, def[ModeIncognito])
	}
	return def[ModeIncognito]
}

func fill(prof, def Profile) Profile {
	if prof.Apps == "" {
		prof.Apps = def.Apps
	}
	if prof.Games == "" {
		prof.Games = def.Games
	}
	if prof.Music == "" {
		prof.Music = def.Music
	}
	if prof.System == "" {
		prof.System = def.System
	}
	if prof.Custom == "" {
		prof.Custom = def.Custom
	}
	return prof
}

type Filter struct {
	policy Policy
	hide   []*regexp.Regexp
}

func New(p Policy) *Filter {
	f := &Filter{policy: p}
	for _, expr := range p.HideApps {
		if re, err := regexp.Compile(expr); err == nil {
			f.hide = append(f.hide, re)
		}
	}
	return f
}

func (f *Filter) Policy() Policy { return f.policy }

func (f *Filter) Apply(snap presence.Snapshot) presence.Snapshot {
	out := snap.Clone()
	prof := f.policy.Active()

	apps := prof.Apps
	if f.hiddenApp(out) {
		apps = LevelOff
	}
	games := prof.Games
	if f.hiddenGame(out) {
		games = LevelOff
	}

	switch apps {
	case LevelApp:
		drop(out, presence.KeyActiveWindow)
	case LevelBusy:
		drop(out, presence.KeyActiveWindow, presence.KeyActiveWorkspace)
		if out.Has(presence.KeyActiveApp) {
			out.Set(presence.KeyActiveApp, BusyLabel)
		}
	case LevelOff:
		drop(out, appKeys...)
	}

	switch games {
	case LevelPlaying:
		drop(out, gameDetailKeys...)
	case LevelOff:
		drop(out, gameKeys...)
	}

	switch prof.Music {
	case LevelListening:
		drop(out, musicDetailKeys...)
	case LevelOff:
		drop(out, musicKeys...)
	}
	// The generic mpris title is whatever any player has open, a browser video
	// included, so it goes with the apps rather than with Spotify.
	if apps != LevelFull {
		drop(out, presence.KeyMusic)
	}

	if prof.System == LevelOff {
		drop(out, systemKeys...)
	}
	if prof.Custom == LevelOff {
		drop(out, out.Strings(presence.KeyCustomFields)...)
		drop(out, presence.KeyCustomFields)
	}
	if f.policy.Hidden() {
		declared := keySet(out.Strings(presence.KeyCustomFields)...)
		for k := range out {
			if !knownKeys[k] && !declared[k] {
				delete(out, k)
			}
		}
	}

	if f.policy.Hidden() && f.policy.Announce {
		out.Set(presence.KeyIncognito, true)
		if f.policy.Note != "" {
			out.Set(presence.KeyIncognitoNote, f.policy.Note)
		}
	}
	return out
}

// hide_apps covers what you never want named, whatever the mode: password
// managers, banking, a title you keep to yourself. The two groups are matched
// apart, so a browser window you would rather not show does not also blank the
// game, and the other way round.
func (f *Filter) hiddenApp(s presence.Snapshot) bool {
	return f.anyMatch(s.String(presence.KeyActiveApp), s.String(presence.KeyActiveWindow))
}

func (f *Filter) hiddenGame(s presence.Snapshot) bool {
	return f.anyMatch(s.String(presence.KeyGameName))
}

func (f *Filter) anyMatch(values ...string) bool {
	for _, re := range f.hide {
		for _, v := range values {
			// A pattern broad enough to match "" must not blank a group that has
			// nothing in it yet
			if v != "" && re.MatchString(v) {
				return true
			}
		}
	}
	return false
}

func drop(s presence.Snapshot, keys ...string) {
	for _, k := range keys {
		delete(s, k)
	}
}

func Path() string { return config.File(FileName) }

func Load() (Policy, error) {
	data, err := config.Read(FileName)
	if err != nil {
		return Default(), nil
	}
	p := Default()
	if err := json.Unmarshal(data, &p); err != nil {
		return Default(), err
	}
	return p, nil
}

func Save(p Policy) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return config.Write(FileName, append(data, '\n'), 0o600)
}

// EnsureConfig writes the defaults out once, so the file is there to be edited
// instead of being a documented secret.
func EnsureConfig() {
	if _, err := config.Read(FileName); err == nil {
		return
	}
	_ = Save(Default())
}

// Set switches the mode, optionally for a while. A duration is what keeps you
// from staying dark for a week without noticing.
func Set(mode string, d time.Duration) (Policy, error) {
	p, _ := Load()
	p.Mode = mode
	p.Until = ""
	if mode != ModeNormal && d > 0 {
		p.Until = time.Now().Add(d).Format(time.RFC3339)
	}
	return p, Save(p)
}

func Toggle() (Policy, error) {
	p, _ := Load()
	if p.Hidden() {
		return Set(ModeNormal, 0)
	}
	return Set(ModeIncognito, 0)
}

func SetNote(note string) (Policy, error) {
	p, _ := Load()
	p.Note = note
	return p, Save(p)
}
