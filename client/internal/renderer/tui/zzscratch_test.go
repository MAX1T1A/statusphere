package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

func TestScratchBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(stats.SpotifyStats{
			TotalSeconds: 13620,
			Daily: []stats.DayStat{{Day: "2026-07-21", Seconds: 0}, {Day: "2026-07-22", Seconds: 0},
				{Day: "2026-07-25", Seconds: 7140}, {Day: "2026-07-27", Seconds: 6420}},
			TopTracks:  []stats.TrackStat{{Title: "Shit I'm Dreaming", Seconds: 720}, {Title: "Madness", Seconds: 540}, {Title: "This Fire", Seconds: 480}},
			TopArtists: []stats.ArtistStat{{Artist: "Oxxxymiron", Seconds: 2040}, {Artist: "King Crimson", Seconds: 780}, {Artist: "Peter Cat Recording Co.", Seconds: 720}},
		})
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel(Options{LocalAccountID: "acc-me", SpotifyCache: stats.NewSpotifyCache(srv.URL, "t", "r")})
	m.spotify.Prime("bob-1")
	m.width, m.height = 126, 40
	m.groups = groupDevices([]presence.Snapshot{
		{presence.KeyDeviceID: "bob-1", presence.KeyAccountID: "acc-bob", presence.KeyAccountName: "Evgeniy", presence.KeyLastSeen: int64(1),
			presence.KeyCPUPercent: 92.0, presence.KeyActiveApp: "ghostty",
			presence.KeySpotifyStatus: "playing", presence.KeySpotifyArtist: "The Verve", presence.KeySpotifyTrack: "Sonnet",
			presence.KeySpotifyAlbum: "Urban Hymns (Remastered 2016)", presence.KeySpotifyDisplay: "The Verve — Sonnet",
			presence.KeySpotifyURI: "spotify:track:x", presence.KeySpotifyPosition: int64(216), presence.KeySpotifyLength: int64(261)},
	}, "acc-me")
	m.clampSelection()
	for _, id := range []string{"cover", "progress", "album", "weekly", "tops"} {
		m.expand.toggle("acc-bob", id)
	}
	fmt.Println("\n===== БАЛАНС КОЛОНОК =====")
	fmt.Println(m.View())
}
