package main

import (
	"time"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/renderer/tui"
)

type noopController struct{}

func (noopController) Nudge(string)        {}
func (noopController) Rename(string)       {}
func (noopController) SyncSpotify(string)  {}
func (noopController) SyncCustom([]string) {}

func main() {
	const localID = "me-device-0001"
	customOrder := []string{"weather", "git"}

	t := tui.New(tui.Options{
		LocalID:     localID,
		CustomOrder: customOrder,
		Controller:  noopController{},
	})

	t.Nudges.Process("alice-0001", "pizza tonight?")
	t.Nudges.ProcessLocal("sure, 8pm")

	now := time.Now().Unix()
	devices := []presence.Snapshot{
		{
			presence.KeyDeviceID:        "alice-0001",
			presence.KeyDeviceName:      "alice",
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
			"weather":                   "⛅ +14°C",
			"git":                       "feature/tui",
		},
		{
			presence.KeyDeviceID:       "bob-0002",
			presence.KeyDeviceName:     "bob",
			presence.KeyLastSeen:       now - 25,
			presence.KeyUptimeHours:    52.0,
			presence.KeyActiveApp:      "Firefox",
			presence.KeyActiveWindow:   "GitHub — pull requests",
			presence.KeySpotifyStatus:  "paused",
			presence.KeySpotifyDisplay: "Aphex Twin — Xtal",
			"weather":                  "☀ +21°C",
			"git":                      "master",
		},
		{
			presence.KeyDeviceID:     localID,
			presence.KeyDeviceName:   "me",
			presence.KeyLastSeen:     now,
			presence.KeyUptimeHours:  0.4,
			presence.KeyActiveApp:    "VS Code",
			presence.KeyActiveWindow: "tui.go — statusphere",
			"weather":                "🌧 +9°C",
			"git":                    "refactor/architecture",
		},
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		t.UpdateDevices(devices)
	}()

	_ = t.Run()
}
