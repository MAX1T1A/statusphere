package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/chat"
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
	memberPoll    = 15 * time.Second
)

type App struct {
	ws       *transport.WSTransport
	watcher  *watcher.Watcher
	feed     *feed.Feed
	custom   *custom.Manager
	notifier *notifier.Notifier

	ui        renderer.Renderer
	chat      *tui.ChatStore
	accountID string
	cfg       *auth.Config

	membersMu     sync.Mutex
	members       []auth.MemberInfo
	memberRefresh chan struct{}
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
		ws:            ws,
		feed:          feed.New(),
		custom:        cm,
		notifier:      notifier.New(cfg.AccountID),
		accountID:     cfg.AccountID,
		cfg:           cfg,
		memberRefresh: make(chan struct{}, 1),
	}
	a.watcher = watcher.New(coll, a.send, watchInterval)

	switch uiMode {
	case "tui":
		t := tui.New(tui.Options{
			SpotifyCache:   stats.NewSpotifyCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
			SummaryCache:   stats.NewSummaryCache(cfg.ServerURL, cfg.Token, "day", cfg.RoomID),
			HourlyCache:    stats.NewHourlyCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
			RoomCache:      stats.NewRoomScreenCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
			RoomID:         cfg.RoomID,
			LocalID:        cfg.DeviceID,
			LocalAccountID: cfg.AccountID,
			Controller:     a,
		})
		a.ui = t
		a.chat = t.Chat
		go a.loadHistory(cfg.ServerURL, cfg.Token, cfg.RoomID)
		go a.pollMembers(ctx)
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

	a.send(coll.Collect(ctx))

	go a.watcher.Run(ctx)
	go a.listen(ctx)
	go a.refresh(ctx)

	return a.ui.Run()
}

func (a *App) send(snap presence.Snapshot) {
	if err := a.ws.Send(snap); err != nil {
		log.Printf("send: %v", err)
	}
	// The server never echoes our own presence back to us, so reflect it into
	// the local feed — otherwise we'd render our own membership row as offline.
	local := snap.Clone()
	local.Set(presence.KeyDeviceID, a.cfg.DeviceID)
	local.Set(presence.KeyAccountID, a.accountID)
	if dn := a.ws.DeviceName(); dn != "" {
		local.Set(presence.KeyDeviceName, dn)
	}
	a.feed.Update(local)
}

func (a *App) listen(ctx context.Context) {
	_ = a.ws.Listen(ctx, func(data []byte) {
		var msg map[string]any
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}

		if kind, _ := msg["type"].(string); kind == "msg" {
			a.handleMessage(msg)
			return
		}

		snap := presence.Snapshot(msg)
		a.feed.Update(snap)
		a.maybeRefreshMembers(snap.String(presence.KeyAccountID))
		a.render()
	})
}

func (a *App) handleMessage(msg map[string]any) {
	from, _ := msg["from"].(string)
	name, _ := msg["from_name"].(string)
	to, _ := msg["to"].(string)
	text, _ := msg["text"].(string)
	at, _ := msg["at"].(string)
	if text == "" {
		return
	}

	if a.chat != nil {
		a.chat.Ingest(from, name, to, text, at)
		a.render()
	}

	if a.accountID != "" && from != a.accountID && to == a.accountID && a.notifier != nil {
		if name == "" {
			name = from
		}
		a.notifier.Handle(from, name, text)
	}
}

func (a *App) loadHistory(serverURL, token, roomID string) {
	if a.chat == nil {
		return
	}
	msgs, err := chat.FetchHistory(serverURL, token, roomID)
	if err != nil {
		log.Printf("chat history: %v", err)
		return
	}
	a.chat.LoadHistory(msgs)
	a.render()
}

func (a *App) refresh(ctx context.Context) {
	ticker := time.NewTicker(refreshRate)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.render()
		}
	}
}

func (a *App) render() {
	a.ui.UpdateDevices(a.roster())
}

// roster merges the canonical room membership with live presence so that
// offline members keep their card; a member is dropped only when the owner
// removes them (server drops them from the member list). Before the first
// member fetch (or if it failed) it falls back to whatever presence is live.
func (a *App) roster() []presence.Snapshot {
	live := a.feed.Snapshot()

	a.membersMu.Lock()
	members := a.members
	a.membersMu.Unlock()

	if len(members) == 0 {
		return live
	}

	byAccount := map[string][]presence.Snapshot{}
	for _, s := range live {
		acc := s.String(presence.KeyAccountID)
		if acc == "" {
			acc = s.DeviceID()
		}
		if acc != "" {
			byAccount[acc] = append(byAccount[acc], s)
		}
	}

	out := make([]presence.Snapshot, 0, len(members))
	for _, m := range members {
		if devs := byAccount[m.AccountID]; len(devs) > 0 {
			for _, d := range devs {
				d.Set(presence.KeyRole, m.Role)
				if d.String(presence.KeyAccountName) == "" && m.Name != "" {
					d.Set(presence.KeyAccountName, m.Name)
				}
			}
			out = append(out, devs...)
			continue
		}
		label := m.Name
		if label == "" {
			label = shortID(m.AccountID)
		}
		out = append(out, presence.Snapshot{
			presence.KeyAccountID:   m.AccountID,
			presence.KeyAccountName: label,
			presence.KeyRole:        m.Role,
			presence.KeyOffline:     true,
		})
	}
	return out
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func (a *App) pollMembers(ctx context.Context) {
	a.refreshMembers()
	ticker := time.NewTicker(memberPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-a.memberRefresh:
		}
		a.refreshMembers()
	}
}

func (a *App) refreshMembers() {
	members, err := a.cfg.Members()
	if err != nil {
		log.Printf("members: %v", err)
		return
	}
	a.membersMu.Lock()
	a.members = members
	a.membersMu.Unlock()
	a.render()
}

// maybeRefreshMembers triggers an out-of-cycle member fetch when presence
// arrives from an account not yet in the cached roster (a fresh join), so new
// members appear promptly instead of waiting for the next poll.
func (a *App) maybeRefreshMembers(accountID string) {
	if accountID == "" {
		return
	}
	a.membersMu.Lock()
	known := len(a.members) == 0
	for _, m := range a.members {
		if m.AccountID == accountID {
			known = true
			break
		}
	}
	a.membersMu.Unlock()
	if !known {
		select {
		case a.memberRefresh <- struct{}{}:
		default:
		}
	}
}

func (a *App) SendMessage(to, text string) {
	if err := a.ws.SendMessage(to, text); err != nil {
		log.Printf("send message: %v", err)
	}
}

func (a *App) Kick(accountID string) {
	if accountID == "" {
		return
	}
	go func() {
		if _, err := a.cfg.Kick(accountID); err != nil {
			log.Printf("kick: %v", err)
			return
		}
		a.refreshMembers()
	}()
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

type ScreenshotOpts struct {
	Device string
	Mode   string
	Width  int
	Height int
	Wait   time.Duration
}

func Screenshot(ctx context.Context, so ScreenshotOpts) (string, error) {
	cfg, err := auth.Load()
	if err != nil {
		return "", fmt.Errorf("no config found: %w", err)
	}

	ws := transport.NewWS(cfg.ServerURL, cfg.Token, cfg.DeviceID, cfg.RoomID)
	if err := ws.Connect(ctx); err != nil {
		return "", fmt.Errorf("connect failed: %w", err)
	}
	defer ws.Close()

	f := feed.New()
	go func() {
		_ = ws.Listen(ctx, func(data []byte) {
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			f.Update(presence.Snapshot(msg))
		})
	}()

	wait := so.Wait
	if wait <= 0 {
		wait = 5 * time.Second
	}
	select {
	case <-ctx.Done():
	case <-time.After(wait):
	}

	devices := f.Snapshot()
	if len(devices) == 0 {
		return "", fmt.Errorf("no devices seen in room within %s", wait)
	}

	target := so.Device
	if target == "" {
		target = pickScreenshotTarget(devices, cfg.AccountID)
	}

	opts := tui.Options{
		SpotifyCache:   stats.NewSpotifyCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
		SummaryCache:   stats.NewSummaryCache(cfg.ServerURL, cfg.Token, "day", cfg.RoomID),
		HourlyCache:    stats.NewHourlyCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
		RoomCache:      stats.NewRoomScreenCache(cfg.ServerURL, cfg.Token, cfg.RoomID),
		RoomID:         cfg.RoomID,
		LocalID:        cfg.DeviceID,
		LocalAccountID: cfg.AccountID,
	}

	mode := so.Mode
	if mode == "" {
		mode = "music"
	}
	width := so.Width
	if width <= 0 {
		width = 100
	}
	height := so.Height
	if height <= 0 {
		height = 32
	}
	return tui.Snapshot(opts, devices, target, mode, width, height), nil
}

func pickScreenshotTarget(devices []presence.Snapshot, ownAccount string) string {
	var fallback string
	for _, d := range devices {
		if d.String(presence.KeyAccountID) == ownAccount {
			continue
		}
		if fallback == "" {
			fallback = d.DeviceID()
		}
		if d.String(presence.KeySpotifyStatus) != "" {
			return d.DeviceID()
		}
	}
	if fallback != "" {
		return fallback
	}
	return devices[0].DeviceID()
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
