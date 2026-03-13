package stats

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

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

type SpotifyCache struct {
	mu        sync.RWMutex
	data      map[string]*entry
	serverURL string
	token     string
}

type entry struct {
	stats     *SpotifyStats
	fetchedAt time.Time
	fetching  bool
}

const cacheTTL = 60 * time.Second

func NewSpotifyCache(serverURL, token string) *SpotifyCache {
	return &SpotifyCache{
		serverURL: serverURL,
		token:     token,
		data:      make(map[string]*entry),
	}
}

func (c *SpotifyCache) Get(deviceID string) *SpotifyStats {
	if deviceID == "" {
		return nil
	}

	c.mu.RLock()
	e, ok := c.data[deviceID]
	if ok && e.stats != nil && time.Since(e.fetchedAt) < cacheTTL {
		result := e.stats
		c.mu.RUnlock()
		return result
	}
	if ok && e.fetching {
		result := e.stats
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	c.mu.Lock()
	if e, ok := c.data[deviceID]; ok && e.fetching {
		c.mu.Unlock()
		return e.stats
	}
	if c.data[deviceID] == nil {
		c.data[deviceID] = &entry{}
	}
	c.data[deviceID].fetching = true
	c.mu.Unlock()

	go c.fetch(deviceID)

	c.mu.RLock()
	if e, ok := c.data[deviceID]; ok {
		result := e.stats
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()
	return nil
}

func (c *SpotifyCache) fetch(deviceID string) {
	u, _ := url.Parse(c.serverURL)
	u.Path = "/stats/spotify"
	q := u.Query()
	q.Set("room_token", c.token)
	q.Set("device_id", deviceID)
	q.Set("period", "week")
	u.RawQuery = q.Encode()

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(u.String())
	if err != nil {
		c.mu.Lock()
		c.data[deviceID].fetching = false
		c.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.mu.Lock()
		c.data[deviceID].fetching = false
		c.mu.Unlock()
		return
	}

	var s SpotifyStats
	if err := json.Unmarshal(body, &s); err != nil {
		c.mu.Lock()
		c.data[deviceID].fetching = false
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.data[deviceID] = &entry{
		stats:     &s,
		fetchedAt: time.Now(),
		fetching:  false,
	}
	c.mu.Unlock()
}
