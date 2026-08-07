package watcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

type capture struct {
	mu    sync.Mutex
	sends []presence.Snapshot
}

func (c *capture) onChange(s presence.Snapshot) {
	c.mu.Lock()
	c.sends = append(c.sends, s)
	c.mu.Unlock()
}

func (c *capture) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.sends)
}

func newWatcher(cur *presence.Snapshot, cap *capture) *Watcher {
	provider := collector.Provider{Name: "test", Collect: func(_ context.Context, s presence.Snapshot) error {
		for k, v := range *cur {
			s[k] = v
		}
		return nil
	}}
	w := New(collector.New(provider), cap.onChange, time.Second)
	w.heartbeat = time.Hour
	return w
}

// The filter runs before change detection, so a hidden app switch is not a
// change and turning the filter on is.
func TestFilterDrivesChangeDetection(t *testing.T) {
	cap := &capture{}
	cur := presence.Snapshot{presence.KeyActiveApp: "kitty"}
	w := newWatcher(&cur, cap)
	ctx := context.Background()

	hide := false
	w.SetFilter(func(s presence.Snapshot) presence.Snapshot {
		if hide {
			delete(s, presence.KeyActiveApp)
		}
		return s
	})

	w.Tick(ctx)
	hide = true
	w.Tick(ctx)
	if cap.count() != 2 {
		t.Fatalf("hiding a field should reach the room, got %d sends", cap.count())
	}

	cur[presence.KeyActiveApp] = "firefox"
	w.Tick(ctx)
	if cap.count() != 2 {
		t.Fatalf("switching apps while hidden changes nothing visible, got %d sends", cap.count())
	}
}

func TestTickSendsOnStableChangeAndDedups(t *testing.T) {
	cap := &capture{}
	cur := presence.Snapshot{presence.KeyActiveApp: "kitty"}
	w := newWatcher(&cur, cap)
	ctx := context.Background()

	w.Tick(ctx)
	if cap.count() != 1 {
		t.Fatalf("first tick should send, got %d", cap.count())
	}

	w.Tick(ctx)
	if cap.count() != 1 {
		t.Fatalf("unchanged tick should not send, got %d", cap.count())
	}

	cur[presence.KeyActiveApp] = "firefox"
	w.Tick(ctx)
	if cap.count() != 2 {
		t.Fatalf("stable change should send, got %d", cap.count())
	}
}

func TestTickIgnoresVolatileUntilHeartbeat(t *testing.T) {
	cap := &capture{}
	cur := presence.Snapshot{presence.KeyUptimeHours: 1.0, presence.KeyActiveApp: "kitty"}
	w := newWatcher(&cur, cap)
	w.heartbeat = 20 * time.Millisecond
	ctx := context.Background()

	w.Tick(ctx)
	if cap.count() != 1 {
		t.Fatalf("first tick should send, got %d", cap.count())
	}

	cur[presence.KeyUptimeHours] = 2.0
	w.Tick(ctx)
	if cap.count() != 1 {
		t.Fatalf("volatile-only change should not send before heartbeat, got %d", cap.count())
	}

	time.Sleep(30 * time.Millisecond)
	cur[presence.KeyUptimeHours] = 3.0
	w.Tick(ctx)
	if cap.count() != 2 {
		t.Fatalf("heartbeat should force a send, got %d", cap.count())
	}
}

func TestInjectOnceTriggersImmediateSend(t *testing.T) {
	cap := &capture{}
	cur := presence.Snapshot{presence.KeyActiveApp: "kitty"}
	w := newWatcher(&cur, cap)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	w.InjectOnce(presence.KeyNudge, "hello")

	deadline := time.After(2 * time.Second)
	for {
		if cap.count() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("inject did not trigger a send")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cap.mu.Lock()
	got := cap.sends[0].String(presence.KeyNudge)
	cap.mu.Unlock()
	if got != "hello" {
		t.Fatalf("injected nudge not present, got %q", got)
	}
}

// The session counter climbs every tick on its own, so treating it as a change
// would publish the whole snapshot twice a second for as long as the game runs.
func TestTickIgnoresGameSessionSeconds(t *testing.T) {
	cap := &capture{}
	cur := presence.Snapshot{
		presence.KeyGameStatus:         "playing",
		presence.KeyGameName:           "Red Dead Redemption 2",
		presence.KeyGameSessionSeconds: 10,
	}
	w := newWatcher(&cur, cap)
	w.heartbeat = time.Hour
	ctx := context.Background()

	w.Tick(ctx)
	for i := 12; i < 30; i += 2 {
		cur[presence.KeyGameSessionSeconds] = i
		w.Tick(ctx)
	}
	if cap.count() != 1 {
		t.Fatalf("a ticking session counter published %d times, want 1", cap.count())
	}

	cur[presence.KeyGameName] = "Cyberpunk 2077"
	w.Tick(ctx)
	if cap.count() != 2 {
		t.Fatalf("switching game should send, got %d", cap.count())
	}
}
