package spotify

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

const (
	dest       = "org.mpris.MediaPlayer2.spotify"
	objectPath = "/org/mpris/MediaPlayer2"
	propIface  = "org.freedesktop.DBus.Properties"
	playerProp = "org.mpris.MediaPlayer2.Player"
)

func init() {
	p := &player{}
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "spotify", Collect: p.collect},
		Applies:  collector.When(collector.OnOS("linux"), collector.OnSessionBus()),
	})
}

type player struct {
	mu   sync.Mutex
	conn *dbus.Conn
}

func getProp(ctx context.Context, conn *dbus.Conn, prop string) (dbus.Variant, error) {
	obj := conn.Object(dest, objectPath)
	var result dbus.Variant
	err := obj.CallWithContext(ctx, propIface+".Get", 0, playerProp, prop).Store(&result)
	return result, err
}

func extractMetadata(ctx context.Context, conn *dbus.Conn) (artist, title, album, artURL, trackID string, length int64) {
	v, err := getProp(ctx, conn, "Metadata")
	if err != nil {
		return
	}

	meta, ok := v.Value().(map[string]dbus.Variant)
	if !ok {
		return
	}

	if l, ok := meta["mpris:length"]; ok {
		switch val := l.Value().(type) {
		case int64:
			length = val
		case uint64:
			length = int64(val)
		}
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

	if t, ok := meta["mpris:trackid"]; ok {
		trackID, _ = t.Value().(string)
	}

	return
}

func (p *player) collect(ctx context.Context, snap presence.Snapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		conn, err := dbus.SessionBus()
		if err != nil {
			return err
		}
		p.conn = conn
	}

	v, err := getProp(ctx, p.conn, "PlaybackStatus")
	if err != nil {
		p.conn = nil
		return err
	}

	status, _ := v.Value().(string)
	if status == "" || status == "Stopped" {
		return nil
	}

	artist, title, album, artURL, trackID, length := extractMetadata(ctx, p.conn)
	if title == "" {
		return nil
	}

	snap.Set(presence.KeySpotifyStatus, strings.ToLower(status))
	snap.Set(presence.KeySpotifyTrack, title)
	snap.Set(presence.KeySpotifyArtist, artist)
	snap.Set(presence.KeySpotifyAlbum, album)
	snap.Set(presence.KeySpotifyArtURL, artURL)

	if length > 0 {
		snap.Set(presence.KeySpotifyLength, int(length/1_000_000))
	}
	if pv, err := getProp(ctx, p.conn, "Position"); err == nil {
		if us, ok := pv.Value().(int64); ok && us > 0 {
			snap.Set(presence.KeySpotifyPosition, int(us/1_000_000))
		}
	}

	if trackID != "" {
		if id, ok := strings.CutPrefix(trackID, "/com/spotify/track/"); ok {
			snap.Set(presence.KeySpotifyURI, "spotify:track:"+id)
		}
	}

	if artist != "" {
		snap.Set(presence.KeySpotifyDisplay, fmt.Sprintf("%s — %s", artist, title))
	} else {
		snap.Set(presence.KeySpotifyDisplay, title)
	}
	return nil
}
