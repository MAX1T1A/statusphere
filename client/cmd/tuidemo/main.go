package main

import (
	"time"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/renderer/tui"
)

type noopController struct{}

func (noopController) SendMessage(string, string) {}
func (noopController) Rename(string)              {}
func (noopController) SyncSpotify(string)         {}
func (noopController) SyncCustom([]string)        {}

func main() {
	const localID = "me-device-0001"

	t := tui.New(tui.Options{
		LocalID:        localID,
		LocalAccountID: "acc-me",
		Controller:     noopController{},
	})

	t.Chat.Ingest("acc-alice", "Alice", "", "pizza tonight?", "")
	t.Chat.Ingest("acc-me", "Me", "", "sure, 8pm", "")
	t.Chat.Ingest("acc-bob", "Bob", "acc-me", "ping me when you're free", "")

	now := time.Now().Unix()
	devices := []presence.Snapshot{
		{
			presence.KeyDeviceID:        "alice-0001",
			presence.KeyAccountID:       "acc-alice",
			presence.KeyAccountName:     "Alice",
			presence.KeyDeviceName:      "alice-laptop",
			presence.KeyLastSeen:        now,
			presence.KeyUptimeHours:     5.3,
			presence.KeyCPUPercent:      12.0,
			presence.KeyMemUsedMB:       6800.0,
			presence.KeyMemTotalMB:      16000.0,
			presence.KeyLoadAvg1m:       0.84,
			presence.KeyPackageCount:    int64(1284),
			presence.KeyActiveApp:       "kitty",
			presence.KeyActiveWindow:    "nvim ~/proj/main.go",
			presence.KeyActiveWorkspace: "2",
			presence.KeySpotifyStatus:   "playing",
			presence.KeySpotifyArtist:   "Boards of Canada",
			presence.KeySpotifyTrack:    "Roygbiv",
			presence.KeySpotifyAlbum:    "Music Has the Right to Children",
			presence.KeySpotifyDisplay:  "Boards of Canada — Roygbiv",
			presence.KeySpotifyURI:      "spotify:track:xxxxxxxxxxxxxxxxxxxxxx",
			presence.KeySpotifyPosition: int64(72),
			presence.KeySpotifyLength:   int64(225),
			presence.KeyCustomFields:    []string{"weather", "git"},
			"weather":                   "partly cloudy +14°",
			"git":                       "feature/tui",
		},
		{
			presence.KeyDeviceID:    "alice-phone",
			presence.KeyAccountID:   "acc-alice",
			presence.KeyAccountName: "Alice",
			presence.KeyDeviceName:  "alice-phone",
			presence.KeyLastSeen:    now - 40,
			presence.KeyUptimeHours: 9.0,
			presence.KeyActiveApp:   "Telegram",
		},
		{
			presence.KeyDeviceID:       "bob-0002",
			presence.KeyAccountID:      "acc-bob",
			presence.KeyAccountName:    "Bob",
			presence.KeyDeviceName:     "bob",
			presence.KeyLastSeen:       now - 25,
			presence.KeyUptimeHours:    52.0,
			presence.KeyActiveApp:      "Firefox",
			presence.KeyActiveWindow:   "GitHub — pull requests",
			presence.KeySpotifyStatus:  "paused",
			presence.KeySpotifyDisplay: "Aphex Twin — Xtal",
			presence.KeyCustomFields:   []string{"weather", "git"},
			"weather":                  "sunny +21°",
			"git":                      "master",
		},
		{
			presence.KeyDeviceID:        localID,
			presence.KeyAccountID:       "acc-me",
			presence.KeyAccountName:     "Me",
			presence.KeyDeviceName:      "me",
			presence.KeyLastSeen:        now,
			presence.KeyUptimeHours:     0.4,
			presence.KeyActiveApp:       "VS Code",
			presence.KeyActiveWindow:    "tui.go — statusphere",
			presence.KeySpotifyStatus:   "playing",
			presence.KeySpotifyArtist:   "Boards of Canada",
			presence.KeySpotifyTrack:    "Roygbiv",
			presence.KeySpotifyDisplay:  "Boards of Canada — Roygbiv",
			presence.KeySpotifyURI:      "spotify:track:xxxxxxxxxxxxxxxxxxxxxx",
			presence.KeySpotifyPosition: int64(30),
			presence.KeySpotifyLength:   int64(225),
			presence.KeyCustomFields:    []string{"weather", "git"},
			"weather":                   "rain +9°",
			"git":                       "refactor/architecture",
		},
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		t.UpdateDevices(devices)
	}()

	_ = t.Run()
}
