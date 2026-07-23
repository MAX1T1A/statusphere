package media

import "github.com/godbus/dbus/v5"

const (
	spotifyDest = "org.mpris.MediaPlayer2.spotify"
	playerPath  = "/org/mpris/MediaPlayer2"
	openURI     = "org.mpris.MediaPlayer2.Player.OpenUri"
)

func OpenSpotifyURI(uri string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	obj := conn.Object(spotifyDest, playerPath)
	return obj.Call(openURI, 0, uri).Err
}
