package tui

import (
	"strings"
	"testing"
	"time"

	"statusphere-client/internal/presence"
)

func sampleGroup() deviceGroup {
	now := time.Now().Unix()
	return deviceGroup{
		key: "acc-alice",
		devices: []presence.Snapshot{
			{
				presence.KeyAccountName:    "Alice",
				presence.KeyDeviceName:     "alice-laptop",
				presence.KeyLastSeen:       now,
				presence.KeyUptimeHours:    5.3,
				presence.KeyCPUPercent:     12.0,
				presence.KeyMemUsedMB:      6800.0,
				presence.KeyMemTotalMB:     16000.0,
				presence.KeyLoadAvg1m:      0.84,
				presence.KeyActiveApp:      "kitty",
				presence.KeyActiveWindow:   "nvim ~/proj/main.go",
				presence.KeySpotifyStatus:  "playing",
				presence.KeySpotifyArtist:  "Boards of Canada",
				presence.KeySpotifyTrack:   "Roygbiv",
				presence.KeySpotifyDisplay: "Boards of Canada — Roygbiv",
				presence.KeyCustomFields:   []string{"weather", "git"},
				"weather":                  "⛅ +14°C",
				"git":                      "main",
			},
			{
				presence.KeyDeviceName:  "alice-phone",
				presence.KeyLastSeen:    now - 40,
				presence.KeyUptimeHours: 9.0,
			},
		},
	}
}

func TestRenderCardStructure(t *testing.T) {
	blocks := []Block{BlockSpotify(nil), BlockApp(nil)}
	out := renderCard(sampleGroup(), blocks, BlockCustom(), false, 70)
	for _, want := range []string{"Alice", "alice-laptop", "alice-phone", "kitty", "Roygbiv", "weather"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered card missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "—\n") || strings.Contains(out, " — ─") {
		t.Fatal("card should not contain empty dash fields")
	}
}
