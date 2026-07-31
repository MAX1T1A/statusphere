package collector

import (
	"context"
	"fmt"
	"log"
	"maps"
	"time"

	"statusphere-client/internal/detector"
	"statusphere-client/internal/presence"
)

type CollectFunc func(ctx context.Context, snap presence.Snapshot) error

type Provider struct {
	Name    string
	Collect CollectFunc
}

type Descriptor struct {
	Provider
	Applies func(detector.Context) bool
}

var registry []Descriptor

func Register(d Descriptor) {
	registry = append(registry, d)
}

func Active(ctx detector.Context) []Provider {
	out := make([]Provider, 0, len(registry))
	for _, d := range registry {
		if d.Applies == nil || d.Applies(ctx) {
			out = append(out, d.Provider)
		}
	}
	return out
}

func OnOS(family string) func(detector.Context) bool {
	return func(c detector.Context) bool { return c.OSFamily == family }
}

func OnDistro(distro string) func(detector.Context) bool {
	return func(c detector.Context) bool { return c.Distro == distro }
}

func OnDEWM(dewm string) func(detector.Context) bool {
	return func(c detector.Context) bool { return c.DEWM == dewm }
}

func OnSessionBus() func(detector.Context) bool {
	return func(c detector.Context) bool { return c.SessionBus }
}

func When(preds ...func(detector.Context) bool) func(detector.Context) bool {
	return func(c detector.Context) bool {
		for _, p := range preds {
			if !p(c) {
				return false
			}
		}
		return true
	}
}

const defaultTimeout = 3 * time.Second

type Collector struct {
	providers []Provider
	timeout   time.Duration
}

func New(providers ...Provider) *Collector {
	return &Collector{providers: providers, timeout: defaultTimeout}
}

func (c *Collector) Collect(ctx context.Context) presence.Snapshot {
	snap := presence.New()
	for _, p := range c.providers {
		c.run(ctx, p, snap)
	}
	return snap
}

func (c *Collector) run(ctx context.Context, p Provider, snap presence.Snapshot) {
	pctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	scratch := presence.New()
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()
		done <- p.Collect(pctx, scratch)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("collector %q: %v", p.Name, err)
			return
		}
		maps.Copy(snap, scratch)
	case <-pctx.Done():
		log.Printf("collector %q: timed out", p.Name)
	}
}
