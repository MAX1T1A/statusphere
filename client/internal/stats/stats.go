package stats

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const cacheTTL = 60 * time.Second

type Fetcher interface {
	Path() string
	Query(deviceID string) url.Values
	New() any
}

type entry struct {
	data      any
	fetchedAt time.Time
	fetching  bool
}

type Cache struct {
	mu        sync.RWMutex
	items     map[string]*entry
	serverURL string
	token     string
	fetcher   Fetcher
}

func NewCache(serverURL, token string, f Fetcher) *Cache {
	return &Cache{
		serverURL: serverURL,
		token:     token,
		fetcher:   f,
		items:     make(map[string]*entry),
	}
}

func (c *Cache) Get(deviceID string) any {
	if deviceID == "" {
		return nil
	}

	c.mu.RLock()
	e, ok := c.items[deviceID]
	if ok && e.data != nil && time.Since(e.fetchedAt) < cacheTTL {
		result := e.data
		c.mu.RUnlock()
		return result
	}
	if ok && e.fetching {
		result := e.data
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	c.mu.Lock()
	if e, ok := c.items[deviceID]; ok && e.fetching {
		c.mu.Unlock()
		return e.data
	}
	if c.items[deviceID] == nil {
		c.items[deviceID] = &entry{}
	}
	c.items[deviceID].fetching = true
	c.mu.Unlock()

	go c.fetch(deviceID)

	c.mu.RLock()
	if e, ok := c.items[deviceID]; ok {
		result := e.data
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()
	return nil
}

func (c *Cache) GetSync(deviceID string) any {
	result, _ := c.doFetch(deviceID)
	return result
}

func (c *Cache) fetch(deviceID string) {
	result, err := c.doFetch(deviceID)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.items[deviceID].fetching = false
		return
	}

	c.items[deviceID] = &entry{
		data:      result,
		fetchedAt: time.Now(),
		fetching:  false,
	}
}

func (c *Cache) doFetch(deviceID string) (any, error) {
	u, _ := url.Parse(c.serverURL)
	u.Path = c.fetcher.Path()

	q := c.fetcher.Query(deviceID)
	q.Set("room_token", c.token)
	u.RawQuery = q.Encode()

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := c.fetcher.New()
	if err := json.Unmarshal(body, result); err != nil {
		return nil, err
	}

	return result, nil
}
