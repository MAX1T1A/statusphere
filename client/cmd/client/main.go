package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/feed"
	"statusphere-client/internal/models"
	"statusphere-client/internal/renderer"
	"statusphere-client/internal/stats"

	linuxc "statusphere-client/internal/collector/linux"
	spotifyc "statusphere-client/internal/collector/linux/spotify"

	archc "statusphere-client/internal/collector/linux/arch"
	hyprlandc "statusphere-client/internal/collector/linux/hyprland"

	"statusphere-client/internal/detector"
	"statusphere-client/internal/notifier"
	"statusphere-client/internal/renderer/noop"
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

var (
	uiMode = flag.String("ui", "tui", "UI mode: tui, headless")
)

func buildProviders(ctx detector.Context) []collector.Provider {
	var providers []collector.Provider

	switch ctx.OSFamily {
	case "linux":
		providers = append(providers,
			linuxc.Uptime(),
			spotifyc.NowPlaying(),
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
	flag.Parse()

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
	var notify *notifier.Notifier
	var nudges *tui.NudgeHistory

	var ui renderer.Renderer
	switch *uiMode {
	case "tui":
		spotifyCache := stats.NewSpotifyCache(serverURL, roomToken)
		summaryCache := stats.NewSummaryCache(serverURL, roomToken, "day")
		tuiUI := tui.New(spotifyCache, summaryCache, func(message string) {
			w.InjectOnce("nudge_message", message)
			if nudges != nil {
				nudges.ProcessLocal(message)
			}
		}, func(name string) {
			transport.SetName(name)
			ws.SetDeviceName(name)
		}, transport.ID())
		ui = tuiUI

		notify = notifier.New(transport.ID())
		nudges = tuiUI.Nudges
	case "headless":
		noop := noop.NewNoop()
		ui = noop
		notify = notifier.New(transport.ID())
		go func() {
			<-ctx.Done()
			noop.Stop()
		}()
	default:
		log.Fatalf("unknown ui mode: %s", *uiMode)
	}

	go w.Run(ctx)

	go func() {
		ws.Listen(ctx, func(data []byte) {
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}

			if nudge, _ := msg["nudge_message"].(string); nudge != "" {
				devID, _ := msg["device_id"].(string)
				devName, _ := msg["device_name"].(string)
				if notify != nil {
					notify.Handle(devID, devName, nudge)
				}
				if nudges != nil {
					nudges.Process(devID, nudge)
				}
			}

			f.Update(msg)
			ui.UpdateDevices(f.Snapshot())
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
				ui.UpdateDevices(f.Snapshot())
			}
		}
	}()

	if err := ui.Run(); err != nil {
		log.Fatalf("ui error: %v", err)
	}
	cancel()
}
