package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"statusphere-client/internal/config"
)

const (
	artVersion   = 1
	storeTimeout = 10 * time.Second
	storeFloor   = 2 * time.Second // between any two store requests
)

// How the last store lookup went; empty means it has not been asked. The
// distinction that matters is missing (a cacheable "no such app") against errored
// (a network problem, which is not).
const (
	storeOK      = "ok"
	storeMissing = "missing"
	storeErrored = "error"
)

type art struct {
	V          int       `json:"v"`
	AppID      int       `json:"appid"`
	Name       string    `json:"name,omitempty"`
	Type       string    `json:"type,omitempty"`
	HeaderURL  string    `json:"header_url,omitempty"`
	CoverURL   string    `json:"cover_url,omitempty"`
	HeroURL    string    `json:"hero_url,omitempty"`
	LogoURL    string    `json:"logo_url,omitempty"`
	Store      string    `json:"store"`
	Attempts   int       `json:"attempts,omitempty"`
	ResolvedAt time.Time `json:"resolved_at"`
}

// resolver answers with everything Steam's own cache knows straight away and asks
// the store in the background for the rest.
type resolver struct {
	mu      sync.Mutex
	known   map[int]art
	pending map[int]bool
	last    time.Time

	steamRoot string
	dir       string
	client    *http.Client
	now       func() time.Time
}

func newResolver(steamRoot string) *resolver {
	return &resolver{
		known:     make(map[int]art),
		pending:   make(map[int]bool),
		steamRoot: steamRoot,
		dir:       filepath.Join(config.CacheDir(), "steam", "art"),
		client:    &http.Client{Timeout: 15 * time.Second},
		now:       time.Now,
	}
}

func (r *resolver) lookup(appID int) art {
	r.mu.Lock()
	a, ok := r.known[appID]
	if !ok {
		if cached, err := r.read(appID); err == nil {
			a, ok = cached, true
		}
	}
	if !ok {
		a = r.fromSteamCache(appID)
	}
	r.known[appID] = a

	start := !r.pending[appID] && r.stale(a)
	if start {
		r.pending[appID] = true
	}
	r.mu.Unlock()

	if start {
		go r.askStore(appID)
	}
	return a
}

// Steam's own cache holds the exact paths its library uses, hashed ones included,
// and reading it costs a quarter of a millisecond - so it happens inline rather
// than on the goroutine. The first tick of a game already has a picture, and
// --published, which is one shot and exits, is not left waiting on anything.
func (r *resolver) fromSteamCache(appID int) art {
	a := art{V: artVersion, AppID: appID}
	paths, ok := localAssets(r.steamRoot, appID)
	if !ok {
		return a
	}
	a.HeaderURL = assetURL(appID, paths.header)
	a.CoverURL = assetURL(appID, paths.capsule)
	a.HeroURL = assetURL(appID, paths.hero)
	a.LogoURL = assetURL(appID, paths.logo)
	return a
}

// A resolved title is never refetched: its art does not move, and a stable url is
// what lets a friend's client cache the picture. Failures widen out to 64h so
// neither a rate limit nor a title that is genuinely not on the store can turn
// into a request per tick.
func (r *resolver) stale(a art) bool {
	age := r.now().Sub(a.ResolvedAt)
	switch a.Store {
	case storeOK:
		return false
	case storeMissing:
		return age > 30*24*time.Hour
	default:
		return age > time.Duration(1<<min(a.Attempts, 6))*time.Hour
	}
}

// The pictures are already in hand by the time this runs, so what the store adds is
// the published title and the type - the only reliable way to tell a tool or a piece
// of dlc from something someone is actually playing.
func (r *resolver) askStore(appID int) {
	defer func() {
		r.mu.Lock()
		delete(r.pending, appID)
		r.mu.Unlock()
	}()

	r.mu.Lock()
	a := r.known[appID]
	r.mu.Unlock()

	r.waitTurn()
	ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
	d, err := fetchStore(ctx, r.client, appID)
	cancel()

	switch {
	case err != nil:
		a.Store, a.Attempts = storeErrored, a.Attempts+1
		log.Printf("steam: appdetails %d: %v", appID, err)
	case d == nil:
		a.Store, a.Attempts = storeMissing, a.Attempts+1
	default:
		a.Store, a.Name, a.Type = storeOK, d.Name, d.Type
		if d.HeaderImage != "" {
			// Carries a ?t= cache buster and is right for every title, so it wins
			// over the one built from appinfo
			a.HeaderURL = d.HeaderImage
		}
	}

	a.ResolvedAt = r.now()
	r.mu.Lock()
	r.known[appID] = a
	r.mu.Unlock()
	r.write(a)
}

func (r *resolver) waitTurn() {
	r.mu.Lock()
	wait := storeFloor - r.now().Sub(r.last)
	r.last = r.now().Add(max(0, wait))
	r.mu.Unlock()
	if wait > 0 {
		time.Sleep(wait)
	}
}

func (r *resolver) path(appID int) string {
	return filepath.Join(r.dir, strconv.Itoa(appID)+".json")
}

func (r *resolver) read(appID int) (art, error) {
	data, err := os.ReadFile(r.path(appID))
	if err != nil {
		return art{}, err
	}
	var a art
	if err := json.Unmarshal(data, &a); err != nil {
		return art{}, err
	}
	if a.V != artVersion || a.AppID != appID {
		return art{}, fmt.Errorf("steam: stale art cache for %d", appID)
	}
	return a, nil
}

func (r *resolver) write(a art) {
	if err := os.MkdirAll(r.dir, 0o700); err != nil {
		return
	}
	data, err := json.Marshal(a)
	if err != nil {
		return
	}
	// The tui and the widget's --ui json process are two clients on one device, so
	// a half-written file has to be impossible rather than unlikely.
	tmp := r.path(a.AppID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	if err := os.Rename(tmp, r.path(a.AppID)); err != nil {
		os.Remove(tmp)
	}
}

const storeEndpoint = "https://store.steampowered.com/api/appdetails"

type storeData struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	HeaderImage string `json:"header_image"`
}

// A nil result with a nil error is the store saying it has no such app, which is a
// cacheable answer. An error is a transport or rate-limit problem, which is not.
func fetchStore(ctx context.Context, c *http.Client, appID int) (*storeData, error) {
	id := strconv.Itoa(appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		storeEndpoint+"?appids="+id+"&filters=basic&l=english", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("appdetails: status %d", resp.StatusCode)
	}
	// filters=basic is advisory, the endpoint still returns the full description
	return decodeStore(io.LimitReader(resp.Body, 4<<20), id)
}

func decodeStore(r io.Reader, id string) (*storeData, error) {
	var body map[string]struct {
		Success bool      `json:"success"`
		Data    storeData `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return nil, err
	}
	e, ok := body[id]
	if !ok || !e.Success {
		return nil, nil
	}
	return &e.Data, nil
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
