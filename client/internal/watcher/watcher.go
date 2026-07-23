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
}

type Watcher struct {
	collector *collector.Collector
	onChange  func(presence.Snapshot)
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
			w.tick(ctx)
		case <-w.trigger:
			w.tick(ctx)
		}
	}
}

func (w *Watcher) tick(ctx context.Context) {
	snap := w.collector.Collect(ctx)

	w.injectMu.Lock()
	maps.Copy(snap, w.inject)
	w.inject = make(map[string]any)
	w.injectMu.Unlock()

	changed := w.last == nil || !w.last.EqualExcept(snap, volatileKeys)
	heartbeatDue := time.Since(w.lastSent) >= w.heartbeat

	if changed || heartbeatDue {
		w.last = snap
		w.lastSent = time.Now()
		w.onChange(snap)
	}
}
