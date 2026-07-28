// Package photo tracks each room member's current shared photo status: one
// entry per account, replaced by the next share, pruned once it expires.
package photo

import (
	"sync"
	"time"
)

type Photo struct {
	AccountID string
	PhotoID   string
	LocalPath string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Store struct {
	mu     sync.Mutex
	photos map[string]Photo
}

func New() *Store {
	return &Store{photos: make(map[string]Photo)}
}

func (s *Store) Update(p Photo) {
	if p.AccountID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.photos[p.AccountID] = p
}

// Snapshot returns the still-live photos, pruning any that have expired.
func (s *Store) Snapshot() []Photo {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Photo, 0, len(s.photos))
	for id, p := range s.photos {
		if now.After(p.ExpiresAt) {
			delete(s.photos, id)
			continue
		}
		out = append(out, p)
	}
	return out
}
