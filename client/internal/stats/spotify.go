package stats

import "net/url"

type DayStat struct {
	Day     string `json:"day"`
	Seconds int    `json:"seconds"`
}

type TrackStat struct {
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Seconds int    `json:"seconds"`
}

type ArtistStat struct {
	Artist  string `json:"artist"`
	Seconds int    `json:"seconds"`
}

type SpotifyStats struct {
	DeviceID     string       `json:"device_id"`
	Period       string       `json:"period"`
	Since        string       `json:"since"`
	TotalSeconds int          `json:"total_seconds"`
	Daily        []DayStat    `json:"daily"`
	TopTracks    []TrackStat  `json:"top_tracks"`
	TopArtists   []ArtistStat `json:"top_artists"`
}

type spotifyFetcher struct {
	period string
	room   string
}

func (f spotifyFetcher) Path() string { return "/stats/spotify" }

func (f spotifyFetcher) Query(deviceID string) url.Values {
	q := url.Values{}
	q.Set("room", f.room)
	q.Set("device_id", deviceID)
	q.Set("period", f.period)
	return q
}

func (f spotifyFetcher) New() any { return &SpotifyStats{} }

func NewSpotifyCache(serverURL, token, room string) *Cache {
	return NewCache(serverURL, token, spotifyFetcher{period: "week", room: room})
}
