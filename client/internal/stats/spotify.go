package stats

import "net/url"

type DayStat struct {
	Day     string `json:"day"`
	Seconds int    `json:"seconds"`
}

type SpotifyStats struct {
	DeviceID     string    `json:"device_id"`
	Period       string    `json:"period"`
	Since        string    `json:"since"`
	TotalSeconds int       `json:"total_seconds"`
	Daily        []DayStat `json:"daily"`
}

type spotifyFetcher struct {
	period string
}

func (f spotifyFetcher) Path() string { return "/stats/spotify" }

func (f spotifyFetcher) Query(deviceID string) url.Values {
	q := url.Values{}
	q.Set("device_id", deviceID)
	q.Set("period", f.period)
	return q
}

func (f spotifyFetcher) New() any { return &SpotifyStats{} }

func NewSpotifyCache(serverURL, token string) *Cache {
	return NewCache(serverURL, token, spotifyFetcher{period: "week"})
}
