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

	linuxc "statusphere-client/internal/collector/linux"
	archc "statusphere-client/internal/collector/linux/arch"
	hyprlandc "statusphere-client/internal/collector/linux/hyprland"

	"statusphere-client/internal/detector"
	"statusphere-client/internal/models"
	"statusphere-client/internal/renderer/tui"

	"statusphere-client/internal/transport"
	"statusphere-client/internal/watcher"
)

const (
	watchInterval = 2 * time.Second
	idleTimeout   = 30 * time.Second
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
		log.Fatalf("initial connect failed: %v", err)
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

	t := tui.New()

	go w.Run(ctx)

	go func() {
		ws.Listen(ctx, func(data []byte) {
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			t.Update(msg)
			w.Resume()
			resetIdle()
		})
	}()

	if err := t.Run(); err != nil {
		log.Fatalf("tui error: %v", err)
	}

	cancel()
}
