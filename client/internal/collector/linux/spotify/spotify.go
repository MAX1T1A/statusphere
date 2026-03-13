package spotify

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"

	"statusphere-client/internal/models"
)

const (
	dest       = "org.mpris.MediaPlayer2.spotify"
	objectPath = "/org/mpris/MediaPlayer2"
	propIface  = "org.freedesktop.DBus.Properties"
	playerProp = "org.mpris.MediaPlayer2.Player"
)

func getProp(conn *dbus.Conn, prop string) (dbus.Variant, error) {
	obj := conn.Object(dest, objectPath)
	var result dbus.Variant
	err := obj.Call(propIface+".Get", 0, playerProp, prop).Store(&result)
	return result, err
}

func extractMetadata(conn *dbus.Conn) (artist, title, album, artURL string) {
	v, err := getProp(conn, "Metadata")
	if err != nil {
		return
	}

	meta, ok := v.Value().(map[string]dbus.Variant)
	if !ok {
		return
	}

	if t, ok := meta["xesam:title"]; ok {
		title, _ = t.Value().(string)
	}

	if a, ok := meta["xesam:artist"]; ok {
		switch val := a.Value().(type) {
		case []string:
			artist = strings.Join(val, ", ")
		case []any:
			var parts []string
			for _, v := range val {
				if s, ok := v.(string); ok {
					parts = append(parts, s)
				}
			}
			artist = strings.Join(parts, ", ")
		case string:
			artist = val
		}
	}

	if a, ok := meta["xesam:album"]; ok {
		album, _ = a.Value().(string)
	}

	if u, ok := meta["mpris:artUrl"]; ok {
		artURL, _ = u.Value().(string)
	}

	return
}

func NowPlaying() func(models.Snapshot) {
	var conn *dbus.Conn

	return func(snap models.Snapshot) {
		if conn == nil {
			var err error
			conn, err = dbus.SessionBus()
			if err != nil {
				return
			}
		}

		v, err := getProp(conn, "PlaybackStatus")
		if err != nil {
			conn = nil
			return
		}

		status, _ := v.Value().(string)
		if status == "" || status == "Stopped" {
			return
		}

		artist, title, album, artURL := extractMetadata(conn)
		if title == "" {
			return
		}

		snap["spotify_status"] = strings.ToLower(status)
		snap["spotify_track"] = title
		snap["spotify_artist"] = artist
		snap["spotify_album"] = album
		snap["spotify_art_url"] = artURL

		if artist != "" {
			snap["spotify_display"] = fmt.Sprintf("%s — %s", artist, title)
		} else {
			snap["spotify_display"] = title
		}
	}
}
