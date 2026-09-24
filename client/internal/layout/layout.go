// Package layout carries the Quickshell widget editor's tile arrangement to
// the room: the client does not know what a tile is, only that
// ~/.config/statusphere/layout.json holds a JSON object to pass along.
package layout

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

const FileName = "layout.json"

const maxSize = 16 * 1024

type Store struct {
	mu      sync.Mutex
	watched config.Watched
	obj     map[string]any
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

	if s.watched.Path == "" {
		s.watched.Path = config.File(FileName)
		s.watched.MaxSize = maxSize
	}

	changed, err := s.watched.Changed()
	if err != nil {
		s.obj = nil
		return nil
	}
	if !changed {
		return s.obj
	}

	data, err := s.watched.Read()
	if err != nil {
		s.obj = nil
		if errors.Is(err, config.ErrTooLarge) {
			log.Printf("event=layout_too_large size=%d", s.watched.Size())
		} else {
			log.Printf("event=layout_read_failed reason=%q", err)
		}
		return nil
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Printf("event=layout_parse_failed reason=%q", err)
		s.obj = nil
		return nil
	}
	if obj == nil {
		log.Printf("event=layout_parse_failed reason=%q", "not a JSON object")
		s.obj = nil
		return nil
	}

	s.obj = obj
	return obj
}
