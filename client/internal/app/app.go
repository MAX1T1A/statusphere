package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/chat"
	"statusphere-client/internal/collector"
	"statusphere-client/internal/collector/custom"
	"statusphere-client/internal/config"
	"statusphere-client/internal/detector"
	"statusphere-client/internal/feed"
	"statusphere-client/internal/health"
	"statusphere-client/internal/layout"
	"statusphere-client/internal/media"
	"statusphere-client/internal/notifier"
	"statusphere-client/internal/photo"
	"statusphere-client/internal/presence"
	"statusphere-client/internal/privacy"
	"statusphere-client/internal/renderer"
	"statusphere-client/internal/renderer/jsonline"
	"statusphere-client/internal/renderer/noop"
	"statusphere-client/internal/renderer/tui"
	"statusphere-client/internal/stats"
	"statusphere-client/internal/transport"
	"statusphere-client/internal/watcher"

	_ "statusphere-client/internal/collector/linux"
	_ "statusphere-client/internal/collector/linux/arch"
	_ "statusphere-client/internal/collector/linux/hyprland"
	_ "statusphere-client/internal/collector/linux/spotify"
	_ "statusphere-client/internal/collector/linux/steam"
)

const (
	watchInterval = 2 * time.Second
	refreshRate   = 1 * time.Second
	photoPoll     = 15 * time.Second
)

type App struct {
	ws       *transport.WSTransport
	watcher  *watcher.Watcher
	feed     *feed.Feed
	roster   *feed.Roster
	custom   *custom.Manager
	notifier *notifier.Notifier

	ui        renderer.Renderer
	chat      *tui.ChatStore
	jsonSink  *jsonline.JSONLine
	photos    *photo.Store
	accountID string
	cfg       *auth.Config
}

// Options is how the process was started. Interval is the collect-and-publish
// period: a desktop wants it short so the room follows what you do, a headless
// box that only reports metrics wants it long so it stops burning cycles.
type Options struct {
	UI       string
	Interval time.Duration
}

func Run(ctx context.Context, opts Options) error {
	cfg, err := auth.Load()
	if err != nil {
		return fmt.Errorf("no config found; register first:\n"+
			"  statusphere --register https://your-server.com\n%w", err)
	}
	if cfg.RoomID == "" {
		return fmt.Errorf("not in a room; join one first:\n  statusphere --join <invite>")
	}

	setupLogging()

	interval := opts.Interval
	if interval <= 0 {
		interval = watchInterval
	}

	coll, cm := newCollector(cfg.Kind)

	ws := transport.NewWS(cfg.ServerURL, cfg.Token, cfg.DeviceID, cfg.RoomID)
	if err := ws.Connect(ctx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer ws.Close()

	a := &App{
		ws:        ws,
		feed:      feed.New(),
		roster:    feed.NewRoster(cfg.Members),
		custom:    cm,
		notifier:  notifier.New(cfg.AccountID),
		photos:    photo.New(),
		accountID: cfg.AccountID,
		cfg:       cfg,
	}
	a.watcher = watcher.New(coll, a.send, interval)
	a.watcher.SetFilter(annotate)

	switch opts.UI {
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
	case "json":
		j := jsonline.New(os.Stdout)
		a.ui = j
		a.jsonSink = j
		go func() {
			<-ctx.Done()
			j.Stop()
		}()
		go a.pollMembers(ctx)
		go a.pollPhotos(ctx)
	default:
		return fmt.Errorf("unknown ui mode: %s", opts.UI)
	}

	a.watcher.Tick(ctx)

	go a.watcher.Run(ctx)
	go a.listen(ctx)
	go a.refresh(ctx)

	return a.ui.Run()
}

func newCollector(kind string) (*collector.Collector, *custom.Manager) {
	sysCtx := detector.Detect()
	custom.EnsureConfig()
	privacy.EnsureConfig()
	health.EnsureConfig()
	cm := custom.Load()

	providers := collector.Active(sysCtx)
	providers = append(providers, cm.Providers()...)
	providers = append(providers, cm.FieldsProvider())
	if kind != "" && kind != presence.KindDesktop {
		providers = append(providers, kindProvider(kind))
	}
	return collector.New(providers...), cm
}

// annotate is the filter chain applied to every snapshot before it leaves
// the device: layout and health add to it, then privacy decides what of the
// result actually ships. Health is judged before the privacy filter runs, so
// hiding the numbers also hides the verdict drawn from them.
func annotate(snap presence.Snapshot) presence.Snapshot {
	return privacy.Shared().Apply(health.Shared().Annotate(layout.Shared().Annotate(snap)))
}

func kindProvider(kind string) collector.Provider {
	return collector.Provider{
		Name: "kind",
		Collect: func(_ context.Context, snap presence.Snapshot) error {
			snap.Set(presence.KeyKind, kind)
			return nil
		},
	}
}

// Published renders the snapshot this device would send right now, after the
// privacy filter. It is the answer to "prove that you're not sending it".
func Published(ctx context.Context) (string, error) {
	kind := ""
	if cfg, err := auth.Load(); err == nil {
		kind = cfg.Kind
	}
	coll, _ := newCollector(kind)
	snap := annotate(coll.Collect(ctx))
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) send(snap presence.Snapshot) {
	if err := a.ws.Send(snap); err != nil {
		log.Printf("send: %v", err)
	}
	a.feed.UpdateOwn(snap, a.cfg.DeviceID, a.accountID, a.ws.DeviceName())
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

		if kind, _ := msg["type"].(string); kind == "photo_status" {
			a.handlePhotoStatus(msg)
			return
		}

		snap := presence.Snapshot(msg)
		a.feed.Update(snap)
		a.roster.Seen(snap.String(presence.KeyAccountID))
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
	a.ui.UpdateDevices(a.roster.Merge(a.feed.Snapshot()))
	if a.jsonSink != nil {
		a.jsonSink.UpdatePhotos(photoOutputs(a.photos.Snapshot()))
	}
}

func photoOutputs(photos []photo.Photo) []jsonline.PhotoOut {
	out := make([]jsonline.PhotoOut, 0, len(photos))
	for _, p := range photos {
		out = append(out, jsonline.PhotoOut{
			AccountID: p.AccountID,
			Path:      "file://" + p.LocalPath,
			CreatedAt: p.CreatedAt.Format(time.RFC3339),
			ExpiresAt: p.ExpiresAt.Format(time.RFC3339),
		})
	}
	return out
}

func (a *App) pollMembers(ctx context.Context) {
	a.roster.Poll(ctx, a.membersRefreshed)
}

func (a *App) membersRefreshed(err error) {
	if err != nil {
		log.Printf("members: %v", err)
		return
	}
	a.render()
}

// handlePhotoStatus reacts to a live "photo_status" broadcast: another device
// (or this one) just shared a new photo. Fetching and caching it happens off
// the WS read loop since it's a network round-trip plus disk I/O.
func (a *App) handlePhotoStatus(msg map[string]any) {
	if a.jsonSink == nil {
		return
	}
	accountID, _ := msg["account_id"].(string)
	photoID, _ := msg["photo_id"].(string)
	createdAt, _ := msg["created_at"].(string)
	expiresAt, _ := msg["expires_at"].(string)
	go a.cachePhoto(accountID, photoID, createdAt, expiresAt)
}

func (a *App) pollPhotos(ctx context.Context) {
	a.refreshPhotos()
	ticker := time.NewTicker(photoPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		a.refreshPhotos()
	}
}

// refreshPhotos seeds the room's current shares on startup/reconnect, closing
// the "I just connected, is a photo already live" gap the WS push alone leaves.
func (a *App) refreshPhotos() {
	list, err := a.cfg.ListRoomPhotos()
	if err != nil {
		log.Printf("photos: %v", err)
		return
	}
	for _, p := range list {
		a.cachePhoto(p.AccountID, p.PhotoID, p.CreatedAt, p.ExpiresAt)
	}
}

// cachePhoto fetches a photo's bytes to a content-addressed local file (unless
// already cached) and records it in a.photos. The content-addressed filename
// (account+photo id) is what makes Quickshell's path-keyed thumbnail cache bust
// correctly on every new share, instead of silently keeping a stale image.
func (a *App) cachePhoto(accountID, photoID, createdAt, expiresAt string) {
	if accountID == "" || photoID == "" {
		return
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(expires) {
		return
	}

	dir := filepath.Join(config.CacheDir(), "photos")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("photo cache dir: %v", err)
		return
	}
	localName := accountID + "-" + photoID + ".jpg"
	localPath := filepath.Join(dir, localName)

	if _, err := os.Stat(localPath); err != nil {
		data, err := auth.FetchPhotoBlob(a.cfg.ServerURL, a.cfg.Token, a.cfg.RoomID, accountID, photoID)
		if err != nil {
			log.Printf("photo fetch: %v", err)
			return
		}
		if err := os.WriteFile(localPath, data, 0o600); err != nil {
			log.Printf("photo write: %v", err)
			return
		}
		prunePhotoFiles(dir, accountID, localName)
	}

	a.photos.Update(photo.Photo{
		AccountID: accountID,
		PhotoID:   photoID,
		LocalPath: localPath,
		CreatedAt: created,
		ExpiresAt: expires,
	})
	a.render()
}

// prunePhotoFiles removes an account's older cached photos once a new one has
// replaced it - a share is a single "current" item, not a history.
func prunePhotoFiles(dir, accountID, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := accountID + "-"
	for _, e := range entries {
		name := e.Name()
		if name == keep || !strings.HasPrefix(name, prefix) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
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
		a.membersRefreshed(a.roster.Refresh())
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
