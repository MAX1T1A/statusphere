package privacy

import (
	"os"
	"sync"
	"time"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

// recheck bounds how often the policy file is stat'ed. A toggle has to reach a
// running client within a tick or two: the tui and the widget's `--ui json`
// process are separate clients on the same device and share only this file.
const recheck = time.Second

type Store struct {
	mu      sync.Mutex
	filter  *Filter
	mod     time.Time
	checked time.Time
}

var shared = &Store{}

func Shared() *Store { return shared }

func (s *Store) Filter() *Filter {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if s.filter != nil && now.Sub(s.checked) < recheck {
		return s.filter
	}
	s.checked = now

	mod := time.Time{}
	if info, err := os.Stat(config.File(FileName)); err == nil {
		mod = info.ModTime()
	}
	if s.filter != nil && mod.Equal(s.mod) {
		return s.filter
	}

	p, _ := Load()
	s.filter = New(p)
	s.mod = mod
	return s.filter
}

func (s *Store) Policy() Policy { return s.Filter().Policy() }

func (s *Store) Apply(snap presence.Snapshot) presence.Snapshot {
	return s.Filter().Apply(snap)
}
