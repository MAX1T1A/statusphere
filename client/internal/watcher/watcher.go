package watcher

import (
	"context"
	"maps"
	"sync"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

const defaultHeartbeat = 30 * time.Second

var volatileKeys = map[string]bool{
	presence.KeyUptimeHours: true,
	presence.KeyCPUPercent:  true,
	presence.KeyMemUsedMB:   true,
	presence.KeyMemTotalMB:  true,
	presence.KeyLoadAvg1m:   true,

	// A counter that climbs on its own would make every collection a change and
	// turn the poll into a publish. game_started_at next to it does not move, so a
	// client can still run a live timer off it.
	presence.KeyGameSessionSeconds: true,
}

type Watcher struct {
	collector *collector.Collector
	onChange  func(presence.Snapshot)
	filter    func(presence.Snapshot) presence.Snapshot
	interval  time.Duration
	heartbeat time.Duration

	last     presence.Snapshot
	lastSent time.Time

	injectMu sync.Mutex
	inject   map[string]any
	trigger  chan struct{}
}

func New(c *collector.Collector, onChange func(presence.Snapshot), interval time.Duration) *Watcher {
	return &Watcher{
		collector: c,
		onChange:  onChange,
		interval:  interval,
		heartbeat: defaultHeartbeat,
		inject:    make(map[string]any),
		trigger:   make(chan struct{}, 1),
	}
}

// SetFilter installs the privacy filter. It runs before change detection, so
// what the watcher compares - and resends - is what the room actually sees:
// flipping incognito is itself a change, and hiding a field does not keep
// resending the same visible snapshot.
func (w *Watcher) SetFilter(f func(presence.Snapshot) presence.Snapshot) {
	w.filter = f
}

func (w *Watcher) InjectOnce(key string, value any) {
	w.injectMu.Lock()
	w.inject[key] = value
	w.injectMu.Unlock()

	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Tick(ctx)
		case <-w.trigger:
			w.Tick(ctx)
		}
	}
}

func (w *Watcher) Tick(ctx context.Context) {
	snap := w.collector.Collect(ctx)

	w.injectMu.Lock()
	maps.Copy(snap, w.inject)
	w.inject = make(map[string]any)
	w.injectMu.Unlock()

	if w.filter != nil {
		snap = w.filter(snap)
	}

	changed := w.last == nil || !w.last.EqualExcept(snap, volatileKeys)
	heartbeatDue := time.Since(w.lastSent) >= w.heartbeat

	if changed || heartbeatDue {
		w.last = snap
		w.lastSent = time.Now()
		w.onChange(snap)
	}
}
