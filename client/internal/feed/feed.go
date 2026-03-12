package feed

import (
	"maps"
	"sync"
	"time"
)

type Device struct {
	Data     map[string]any
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

func (f *Feed) Update(data map[string]any) {
	id, ok := data["device_id"].(string)
	if !ok {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.devices[id] = &Device{
		Data:     data,
		LastSeen: time.Now(),
	}
}

func (f *Feed) Snapshot() []map[string]any {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make([]map[string]any, 0, len(f.devices))
	for _, dev := range f.devices {
		out := make(map[string]any, len(dev.Data)+1)
		maps.Copy(out, dev.Data)
		out["last_seen"] = dev.LastSeen.Unix()
		result = append(result, out)
	}
	return result
}
