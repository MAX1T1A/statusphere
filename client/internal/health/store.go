package health

import (
	"os"
	"sync"
	"time"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

// recheck bounds how often the thresholds file is stat'ed. Editing it over ssh
// on a box that is already unhappy should take effect without a restart.
const recheck = 5 * time.Second

type Store struct {
	mu         sync.Mutex
	thresholds Thresholds
	loaded     bool
	mod        time.Time
	checked    time.Time
}

var shared = &Store{}

func Shared() *Store { return shared }

func (s *Store) Thresholds() Thresholds {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if s.loaded && now.Sub(s.checked) < recheck {
		return s.thresholds
	}
	s.checked = now

	mod := time.Time{}
	if info, err := os.Stat(config.File(FileName)); err == nil {
		mod = info.ModTime()
	}
	if s.loaded && mod.Equal(s.mod) {
		return s.thresholds
	}

	s.thresholds = Load()
	s.loaded = true
	s.mod = mod
	return s.thresholds
}

func (s *Store) Annotate(snap presence.Snapshot) presence.Snapshot {
	return s.Thresholds().Annotate(snap)
}
