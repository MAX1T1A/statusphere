package feed

import (
	"context"
	"sync"
	"time"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/presence"
)

const memberPoll = 15 * time.Second

type Roster struct {
	fetch   func() ([]auth.MemberInfo, error)
	refresh chan struct{}

	mu      sync.Mutex
	members []auth.MemberInfo
	labels  map[string]string
}

func NewRoster(fetch func() ([]auth.MemberInfo, error)) *Roster {
	return &Roster{fetch: fetch, refresh: make(chan struct{}, 1)}
}

func (r *Roster) Refresh() error {
	members, err := r.fetch()
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.members = members
	r.mu.Unlock()
	return nil
}

func (r *Roster) Poll(ctx context.Context, done func(error)) {
	done(r.Refresh())
	ticker := time.NewTicker(memberPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-r.refresh:
		}
		done(r.Refresh())
	}
}

func (r *Roster) Seen(accountID string) {
	if accountID == "" {
		return
	}
	r.mu.Lock()
	// Before the first fetch every account is unknown, which is no reason to fetch twice.
	known := len(r.members) == 0
	for _, m := range r.members {
		if m.AccountID == accountID {
			known = true
			break
		}
	}
	r.mu.Unlock()
	if !known {
		r.Kick()
	}
}

// Kick nudges an immediate refresh instead of waiting for the next poll tick.
func (r *Roster) Kick() {
	select {
	case r.refresh <- struct{}{}:
	default:
	}
}

func (r *Roster) Merge(live []presence.Snapshot) []presence.Snapshot {
	r.mu.Lock()
	members := r.members
	r.mu.Unlock()

	if len(members) == 0 {
		return live
	}

	byAccount := map[string][]presence.Snapshot{}
	for _, s := range live {
		acc := s.String(presence.KeyAccountID)
		if acc == "" {
			acc = s.DeviceID()
		}
		if acc != "" {
			byAccount[acc] = append(byAccount[acc], s)
			r.rememberLabel(acc, s)
		}
	}

	out := make([]presence.Snapshot, 0, len(members))
	for _, m := range members {
		if devs := byAccount[m.AccountID]; len(devs) > 0 {
			for _, d := range devs {
				d.Set(presence.KeyRole, m.Role)
				if d.String(presence.KeyAccountName) == "" && m.Name != "" {
					d.Set(presence.KeyAccountName, m.Name)
				}
			}
			out = append(out, devs...)
			continue
		}
		label := m.Name
		if label == "" {
			label = r.lastLabel(m.AccountID)
		}
		if label == "" {
			label = shortID(m.AccountID)
		}
		out = append(out, presence.Snapshot{
			presence.KeyAccountID:   m.AccountID,
			presence.KeyAccountName: label,
			presence.KeyRole:        m.Role,
			presence.KeyOffline:     true,
		})
	}
	return out
}

// rememberLabel keeps the label a device carried while it was online, so its owner
// does not turn into a raw account id the moment the feed drops them.
func (r *Roster) rememberLabel(accountID string, s presence.Snapshot) {
	label := s.String(presence.KeyAccountName)
	if label == "" {
		label = s.DeviceName()
	}
	if label == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.labels == nil {
		r.labels = map[string]string{}
	}
	r.labels[accountID] = label
}

func (r *Roster) lastLabel(accountID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.labels[accountID]
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
