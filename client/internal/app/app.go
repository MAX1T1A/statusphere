package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/collector"
	"statusphere-client/internal/collector/custom"
	"statusphere-client/internal/config"
	"statusphere-client/internal/detector"
	"statusphere-client/internal/feed"
	"statusphere-client/internal/media"
	"statusphere-client/internal/notifier"
	"statusphere-client/internal/presence"
	"statusphere-client/internal/renderer"
	"statusphere-client/internal/renderer/noop"
	"statusphere-client/internal/renderer/tui"
	"statusphere-client/internal/stats"
	"statusphere-client/internal/transport"
	"statusphere-client/internal/watcher"

	_ "statusphere-client/internal/collector/linux"
	_ "statusphere-client/internal/collector/linux/arch"
	_ "statusphere-client/internal/collector/linux/hyprland"
	_ "statusphere-client/internal/collector/linux/spotify"
)

const (
	watchInterval = 2 * time.Second
	refreshRate   = 1 * time.Second
)

type App struct {
	ws       *transport.WSTransport
	watcher  *watcher.Watcher
	feed     *feed.Feed
	custom   *custom.Manager
	notifier *notifier.Notifier

	ui     renderer.Renderer
	nudges *tui.NudgeHistory
}

func Run(ctx context.Context, uiMode string) error {
	cfg, err := auth.Load()
	if err != nil {
		return fmt.Errorf("no config found; register first:\n"+
			"  statusphere --register https://your-server.com\n%w", err)
	}

	setupLogging()

	sysCtx := detector.Detect()
	custom.EnsureConfig()
	cm := custom.Load()

	providers := collector.Active(sysCtx)
	providers = append(providers, cm.Providers()...)
	providers = append(providers, cm.FieldsProvider())
	coll := collector.New(providers...)

	ws := transport.NewWS(cfg.ServerURL, cfg.Token, cfg.DeviceID, cfg.RoomID)
	if err := ws.Connect(ctx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer ws.Close()

	a := &App{
		ws:       ws,
		feed:     feed.New(),
		custom:   cm,
		notifier: notifier.New(cfg.AccountID),
	}
	a.watcher = watcher.New(coll, a.send, watchInterval)

	switch uiMode {
	case "tui":
		t := tui.New(tui.Options{
			SpotifyCache:   stats.NewSpotifyCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
			SummaryCache:   stats.NewSummaryCache(cfg.ServerURL, cfg.Token, "day", cfg.RoomID),
			LocalID:        cfg.DeviceID,
			LocalAccountID: cfg.AccountID,
			Controller:     a,
		})
		a.ui = t
		a.nudges = t.Nudges
	case "headless":
		n := noop.NewNoop()
		a.ui = n
		go func() {
			<-ctx.Done()
			n.Stop()
		}()
	default:
		return fmt.Errorf("unknown ui mode: %s", uiMode)
	}

	if err := ws.Send(coll.Collect(ctx)); err != nil {
		log.Printf("initial send: %v", err)
	}

	go a.watcher.Run(ctx)
	go a.listen(ctx)
	go a.refresh(ctx)

	return a.ui.Run()
}

func (a *App) send(snap presence.Snapshot) {
	if err := a.ws.Send(snap); err != nil {
		log.Printf("send: %v", err)
	}
}

func (a *App) listen(ctx context.Context) {
	_ = a.ws.Listen(ctx, func(data []byte) {
		var msg map[string]any
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		snap := presence.Snapshot(msg)

		if nudge := snap.String(presence.KeyNudge); nudge != "" {
			accountID := snap.String(presence.KeyAccountID)
			name := snap.String(presence.KeyAccountName)
			if name == "" {
				name = snap.DeviceName()
			}
			if a.notifier != nil {
				a.notifier.Handle(accountID, name, nudge)
			}
			if a.nudges != nil {
				a.nudges.Process(accountID, name, nudge)
			}
		}

		a.feed.Update(snap)
		a.ui.UpdateDevices(a.feed.Snapshot())
	})
}

func (a *App) refresh(ctx context.Context) {
	ticker := time.NewTicker(refreshRate)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.ui.UpdateDevices(a.feed.Snapshot())
		}
	}
}

func (a *App) Nudge(message string) {
	a.watcher.InjectOnce(presence.KeyNudge, message)
	if a.nudges != nil {
		a.nudges.ProcessLocal(message)
	}
}

func (a *App) Rename(name string) {
	a.ws.SetDeviceName(name)
}

func (a *App) SyncSpotify(uri string) {
	if err := media.OpenSpotifyURI(uri); err != nil {
		log.Printf("spotify sync: %v", err)
	}
}

func (a *App) SyncCustom(fields []string) {
	a.custom.MergeKeys(fields)
}

func setupLogging() {
	if err := os.MkdirAll(config.CacheDir(), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(config.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(f)
}
