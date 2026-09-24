// Package layout carries the Quickshell widget editor's tile arrangement
// through to the room unread: the client does not know what a tile is, only
// that ~/.config/statusphere/layout.json holds a JSON object to pass along.
package layout

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

const FileName = "layout.json"

const maxSize = 16 * 1024

type Store struct {
	mu        sync.Mutex
	attempted bool
	mod       time.Time
	size      int64
	obj       map[string]any
}

var shared = &Store{}

func Shared() *Store { return shared }

// Annotate adds the parsed layout to the snapshot when the file exists and
// holds a valid JSON object, and leaves the snapshot untouched otherwise.
func (s *Store) Annotate(snap presence.Snapshot) presence.Snapshot {
	if obj := s.resolve(); obj != nil {
		snap.Set(presence.KeyLayout, obj)
	}
	return snap
}

func (s *Store) resolve() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := config.File(FileName)
	info, err := os.Stat(path)
	if err != nil {
		s.attempted = false
		s.obj = nil
		return nil
	}

	if s.attempted && info.ModTime().Equal(s.mod) && info.Size() == s.size {
		return s.obj
	}
	s.attempted = true
	s.mod = info.ModTime()
	s.size = info.Size()
	s.obj = nil

	if info.Size() > maxSize {
		log.Printf("event=layout_too_large size=%d", info.Size())
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("event=layout_read_failed reason=%q", err)
		return nil
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Printf("event=layout_read_failed reason=%q", err)
		return nil
	}
	if obj == nil {
		log.Printf("event=layout_read_failed reason=%q", "not a JSON object")
		return nil
	}

	s.obj = obj
	return obj
}
