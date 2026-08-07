package steam

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	s := &state{}
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "steam", Collect: s.collect},
		Applies:  collector.OnOS("linux"),
	})
}

const (
	rootRecheck = time.Minute
	envRecheck  = 10 * time.Second
	libsRecheck = 5 * time.Minute
)

// Steam calls a demo a demo and a mod a mod, and all of them are someone playing
// something. A tool or a piece of dlc is not.
var playableType = map[string]bool{"game": true, "demo": true, "mod": true, "episode": true}

type state struct {
	mu sync.Mutex

	root        string
	rootChecked time.Time
	libs        []string
	libsChecked time.Time

	boot     time.Time
	cur      session
	lastEnv  time.Time
	names    map[int]string
	resolver *resolver
}

func (s *state) collect(_ context.Context, snap presence.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	root := s.steamRoot()
	if root == "" {
		return nil
	}

	cur, ok := s.session()
	if !ok {
		return nil
	}

	name := s.appName(root, cur.appID)
	if isNonGame(cur.appID, name) {
		return nil
	}

	// The verdict on what this app is comes with the art, so it has to be asked for
	// before anything is written: a tool the name filter missed must not flash up as
	// a game for the tick before the store answers.
	a := s.resolver.lookup(cur.appID)
	if a.Type != "" && !playableType[a.Type] {
		return nil
	}
	if a.Name != "" {
		name = a.Name // the published title beats the install folder's copy
	}
	if name == "" {
		return nil
	}

	snap.Set(presence.KeyGameSource, presence.GameSourceSteam)
	snap.Set(presence.KeyGameStatus, presence.GameStatusPlaying)
	snap.Set(presence.KeyGameAppID, cur.appID)
	snap.Set(presence.KeyGameName, name)
	snap.Set(presence.KeyGameDisplay, name)
	if !cur.started.IsZero() {
		snap.Set(presence.KeyGameStartedAt, cur.started.UTC().Format(time.RFC3339))
		snap.Set(presence.KeyGameSessionSeconds, int(time.Since(cur.started).Seconds()))
	}
	setNonEmpty(snap, presence.KeyGameHeaderURL, a.HeaderURL)
	setNonEmpty(snap, presence.KeyGameCoverURL, a.CoverURL)
	setNonEmpty(snap, presence.KeyGameHeroURL, a.HeroURL)
	setNonEmpty(snap, presence.KeyGameLogoURL, a.LogoURL)
	return nil
}

// Applies is evaluated once at startup, so gating registration on Steam being
// installed would mean installing it needs a client restart. The root is looked up
// lazily instead: one stat a minute on a machine that has no Steam.
func (s *state) steamRoot() string {
	if s.root != "" {
		return s.root
	}
	if time.Since(s.rootChecked) < rootRecheck {
		return ""
	}
	s.rootChecked = time.Now()
	s.root = findRoot()
	if s.root != "" {
		s.names = make(map[int]string)
		s.resolver = newResolver(s.root)
	}
	return s.root
}

// A full walk of /proc every two seconds is wasteful once the game is known, so the
// known pid is re-read first and the walk only runs when that comes up empty. The
// environ pass reads sixteen times as much per process, so it waits its turn.
func (s *state) session() (session, bool) {
	if s.cur.pid != 0 && alive(s.cur) {
		return s.cur, true
	}
	if found, ok := scanProc("cmdline", cmdlineProbe, parseSteamLaunchAppID, s.bootTime()); ok {
		s.cur = found
		return found, true
	}
	if time.Since(s.lastEnv) >= envRecheck {
		s.lastEnv = time.Now()
		if found, ok := scanProc("environ", environProbe, parseSteamAppIDEnv, s.bootTime()); ok {
			s.cur = found
			return found, true
		}
	}
	s.cur = session{}
	return session{}, false
}

func (s *state) bootTime() time.Time {
	if !s.boot.IsZero() {
		return s.boot
	}
	data, err := os.ReadFile(filepath.Join(procRoot, "stat"))
	if err != nil {
		return time.Time{}
	}
	btime, ok := parseBootTime(data)
	if !ok {
		return time.Time{}
	}
	s.boot = time.Unix(btime, 0)
	return s.boot
}

func (s *state) appName(root string, appID int) string {
	if name, ok := s.names[appID]; ok {
		return name
	}
	if time.Since(s.libsChecked) >= libsRecheck || s.libs == nil {
		s.libsChecked = time.Now()
		s.libs = libraryRoots(root)
	}
	found, _ := findApp(s.libs, appID)
	s.names[appID] = found.name
	return found.name
}

func setNonEmpty(snap presence.Snapshot, key, value string) {
	if value != "" {
		snap.Set(key, value)
	}
}
