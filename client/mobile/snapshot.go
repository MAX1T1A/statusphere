package mobile

import (
	"encoding/json"
	"fmt"
	"strings"

	"statusphere-client/internal/presence"
)

type phoneSnapshot struct {
	Music        *phoneMusic   `json:"music"`
	Video        *phoneVideo   `json:"video"`
	App          *phoneApp     `json:"app"`
	Battery      *phoneBattery `json:"battery"`
	AlarmAt      *int64        `json:"alarm_at"`
	MeetingUntil *int64        `json:"meeting_until"`
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

type phoneVideo struct {
	Title           string `json:"title"`
	Channel         string `json:"channel"`
	Status          string `json:"status"`
	PositionSeconds int    `json:"position_seconds"`
	LengthSeconds   int    `json:"length_seconds"`
}

type phoneApp struct {
	Label   string `json:"label"`
	Package string `json:"package"`
}

type phoneBattery struct {
	Percent  int  `json:"percent"`
	Charging bool `json:"charging"`
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
	return in.presence(), nil
}

func (in phoneSnapshot) presence() presence.Snapshot {
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
	if v := in.Video; v != nil && v.Title != "" {
		snap.Set(presence.KeyVideoStatus, strings.ToLower(v.Status))
		snap.Set(presence.KeyVideoTitle, v.Title)
		snap.Set(presence.KeyVideoChannel, v.Channel)
		if v.LengthSeconds > 0 {
			snap.Set(presence.KeyVideoLength, v.LengthSeconds)
		}
		if v.PositionSeconds > 0 {
			snap.Set(presence.KeyVideoPosition, v.PositionSeconds)
		}
	}
	if a := in.App; a != nil && a.Label != "" {
		snap.Set(presence.KeyActiveApp, a.Label)
		if a.Package != "" {
			snap.Set(packageKey, a.Package)
		}
	}
	if b := in.Battery; b != nil {
		snap.Set(presence.KeyBatteryPercent, b.Percent)
		snap.Set(presence.KeyBatteryCharging, b.Charging)
	}
	if at := in.AlarmAt; at != nil {
		snap.Set(presence.KeyAlarmAt, *at)
	}
	if until := in.MeetingUntil; until != nil {
		snap.Set(presence.KeyMeetingUntil, *until)
	}
	return snap
}

func withoutPackage(filtered presence.Snapshot) presence.Snapshot {
	delete(filtered, packageKey)
	return filtered
}
