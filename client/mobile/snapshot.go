package mobile

import (
	"encoding/json"
	"fmt"
	"strings"

	"statusphere-client/internal/presence"
)

type phoneSnapshot struct {
	Music *phoneMusic `json:"music"`
	App   *phoneApp   `json:"app"`
}

type phoneMusic struct {
	Track           string `json:"track"`
	Artist          string `json:"artist"`
	Album           string `json:"album"`
	ArtURL          string `json:"art_url"`
	Status          string `json:"status"`
	PositionSeconds int    `json:"position_seconds"`
	LengthSeconds   int    `json:"length_seconds"`
}

type phoneApp struct {
	Label   string `json:"label"`
	Package string `json:"package"`
}

// packageKey carries the app package through the privacy filter only, so
// hide_apps patterns can match it: the filter looks at active_app and
// active_window, and a phone has no window title to put there.
const packageKey = presence.KeyActiveWindow

func parsePhoneSnapshot(raw string) (presence.Snapshot, error) {
	var in phoneSnapshot
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return nil, fmt.Errorf("snapshot json: %w", err)
	}

	snap := presence.New()
	if m := in.Music; m != nil && m.Track != "" {
		snap.Set(presence.KeySpotifyStatus, strings.ToLower(m.Status))
		snap.Set(presence.KeySpotifyTrack, m.Track)
		snap.Set(presence.KeySpotifyArtist, m.Artist)
		snap.Set(presence.KeySpotifyAlbum, m.Album)
		snap.Set(presence.KeySpotifyArtURL, m.ArtURL)
		snap.Set(presence.KeySpotifyDisplay, presence.SpotifyDisplay(m.Artist, m.Track))
		snap.Set(presence.KeyMusic, presence.SpotifyDisplay(m.Artist, m.Track))
		if m.LengthSeconds > 0 {
			snap.Set(presence.KeySpotifyLength, m.LengthSeconds)
		}
		if m.PositionSeconds > 0 {
			snap.Set(presence.KeySpotifyPosition, m.PositionSeconds)
		}
	}
	if a := in.App; a != nil && a.Label != "" {
		snap.Set(presence.KeyActiveApp, a.Label)
		if a.Package != "" {
			snap.Set(packageKey, a.Package)
		}
	}
	return snap, nil
}

func withoutPackage(filtered presence.Snapshot) presence.Snapshot {
	delete(filtered, packageKey)
	return filtered
}
