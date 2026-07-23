package collector

import (
	"context"
	"errors"
	"testing"
	"time"

	"statusphere-client/internal/detector"
	"statusphere-client/internal/presence"
)

func TestActiveFiltersByPredicate(t *testing.T) {
	saved := registry
	registry = nil
	defer func() { registry = saved }()

	Register(Descriptor{Provider: Provider{Name: "linux-only"}, Applies: OnOS("linux")})
	Register(Descriptor{Provider: Provider{Name: "arch-only"}, Applies: OnDistro("arch")})
	Register(Descriptor{Provider: Provider{Name: "always"}})

	got := Active(detector.Context{OSFamily: "linux", Distro: "debian"})
	names := map[string]bool{}
	for _, p := range got {
		names[p.Name] = true
	}
	if !names["linux-only"] || !names["always"] || names["arch-only"] {
		t.Fatalf("unexpected active set: %v", names)
	}
}

func TestCollectMergesAndIsolatesErrors(t *testing.T) {
	good := Provider{Name: "good", Collect: func(_ context.Context, s presence.Snapshot) error {
		s.Set("a", "1")
		return nil
	}}
	bad := Provider{Name: "bad", Collect: func(_ context.Context, s presence.Snapshot) error {
		s.Set("b", "2")
		return errors.New("boom")
	}}
	c := New(good, bad)
	snap := c.Collect(context.Background())
	if snap.String("a") != "1" {
		t.Fatal("good provider result missing")
	}
	if snap.Has("b") {
		t.Fatal("failed provider must not contribute its scratch data")
	}
}

func TestCollectRecoversFromPanic(t *testing.T) {
	panicky := Provider{Name: "panic", Collect: func(_ context.Context, _ presence.Snapshot) error {
		panic("boom")
	}}
	good := Provider{Name: "good", Collect: func(_ context.Context, s presence.Snapshot) error {
		s.Set("a", "1")
		return nil
	}}
	c := New(panicky, good)
	snap := c.Collect(context.Background())
	if snap.String("a") != "1" {
		t.Fatal("panicking provider must not prevent later providers from running")
	}
}

func TestCollectHonorsTimeout(t *testing.T) {
	c := New(Provider{Name: "hang", Collect: func(ctx context.Context, s presence.Snapshot) error {
		<-ctx.Done()
		return ctx.Err()
	}})
	c.timeout = 20 * time.Millisecond

	start := time.Now()
	snap := c.Collect(context.Background())
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("collect did not respect provider timeout")
	}
	if len(snap) != 0 {
		t.Fatalf("hung provider should contribute nothing, got %v", snap)
	}
}
