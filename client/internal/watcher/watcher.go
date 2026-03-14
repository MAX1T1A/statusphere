package watcher

import (
	"context"
	"sync"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/models"
)

type Watcher struct {
	collector *collector.Collector
	onChange  func(models.Snapshot)
	interval  time.Duration
	last      *models.Snapshot
	paused    bool
	pauseC    chan bool

	injectMu sync.Mutex
	inject   map[string]any
}

func New(c *collector.Collector, onChange func(models.Snapshot), interval time.Duration) *Watcher {
	return &Watcher{
		collector: c,
		onChange:  onChange,
		interval:  interval,
		pauseC:    make(chan bool, 1),
		inject:    make(map[string]any),
	}
}

func (w *Watcher) InjectOnce(key string, value any) {
	w.injectMu.Lock()
	w.inject[key] = value
	w.injectMu.Unlock()
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case p := <-w.pauseC:
			w.paused = p
			if !p {
				w.last = nil
			}
		case <-ticker.C:
			if w.paused {
				continue
			}
			snap := w.collector.Collect()

			w.injectMu.Lock()
			for k, v := range w.inject {
				snap[k] = v
			}
			w.inject = make(map[string]any)
			w.injectMu.Unlock()

			if w.last == nil || !w.last.Equal(snap) {
				w.last = &snap
				w.onChange(snap)
			}
		}
	}
}

func (w *Watcher) Pause() {
	select {
	case w.pauseC <- true:
	default:
	}
}

func (w *Watcher) Resume() {
	select {
	case w.pauseC <- false:
	default:
	}
}
