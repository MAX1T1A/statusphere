package feed

import (
	"sync"
	"time"

	"statusphere-client/internal/presence"
)

const staleTTL = 5 * time.Minute

type Device struct {
	Data     presence.Snapshot
	LastSeen time.Time
}

type Feed struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

func New() *Feed {
	return &Feed{
		devices: make(map[string]*Device),
	}
}

func (f *Feed) Update(data presence.Snapshot) {
	id := data.DeviceID()
	if id == "" {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.devices[id] = &Device{
		Data:     data,
		LastSeen: time.Now(),
	}
}

func (f *Feed) Snapshot() []presence.Snapshot {
	now := time.Now()

	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]presence.Snapshot, 0, len(f.devices))
	for id, dev := range f.devices {
		if now.Sub(dev.LastSeen) > staleTTL {
			delete(f.devices, id)
			continue
		}
		out := dev.Data.Clone()
		out.Set(presence.KeyLastSeen, dev.LastSeen.Unix())
		result = append(result, out)
	}
	return result
}
