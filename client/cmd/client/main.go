package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/feed"
	"statusphere-client/internal/models"

	linuxc "statusphere-client/internal/collector/linux"
	archc "statusphere-client/internal/collector/linux/arch"
	hyprlandc "statusphere-client/internal/collector/linux/hyprland"

	"statusphere-client/internal/detector"
	"statusphere-client/internal/renderer/tui"

	"statusphere-client/internal/transport"
	"statusphere-client/internal/watcher"
)

const (
	watchInterval = 2 * time.Second
	idleTimeout   = 30 * time.Second
	refreshRate   = 1 * time.Second
	serverURL     = "https://sphere.ug3n.com"
	roomToken     = "my-room-token"
)

func buildProviders(ctx detector.Context) []collector.Provider {
	var providers []collector.Provider

	switch ctx.OSFamily {
	case "linux":
		providers = append(providers,
			linuxc.CPUPercent(),
			linuxc.Memory(),
			linuxc.LoadAvg(),
			linuxc.Uptime(),
			linuxc.Music(),
		)

		switch ctx.Distro {
		case "arch":
			providers = append(providers,
				archc.PackageCount(),
			)
		}

		switch ctx.DEWM {
		case "hyprland":
			providers = append(providers,
				hyprlandc.ActiveWindow(),
				hyprlandc.ActiveWorkspace(),
			)
		}
	}

	return providers
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sysCtx := detector.Detect()
	providers := buildProviders(sysCtx)
	coll := collector.New(providers...)

	ws := transport.NewWS(serverURL, roomToken)
	if err := ws.Connect(ctx); err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer ws.Close()

	var sendMu sync.Mutex
	sendSnap := func(snap models.Snapshot) {
		sendMu.Lock()
		defer sendMu.Unlock()
		if err := ws.Send(snap); err != nil {
			log.Printf("send error: %v", err)
		}
	}

	initial := coll.Collect()
	sendSnap(initial)

	w := watcher.New(coll, sendSnap, watchInterval)
	w.Pause()

	var idleTimer *time.Timer
	resetIdle := func() {
		if idleTimer != nil {
			idleTimer.Stop()
		}
		idleTimer = time.AfterFunc(idleTimeout, func() {
			w.Pause()
		})
	}

	f := feed.New()
	t := tui.New()

	go w.Run(ctx)

	go func() {
		ws.Listen(ctx, func(data []byte) {
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			f.Update(msg)
			t.UpdateDevices(f.Snapshot())
			w.Resume()
			resetIdle()
		})
	}()

	go func() {
		ticker := time.NewTicker(refreshRate)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t.UpdateDevices(f.Snapshot())
			}
		}
	}()

	if err := t.Run(); err != nil {
		log.Fatalf("tui error: %v", err)
	}
	cancel()
}
