package steam

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeStore(t *testing.T) {
	const ok = `{"1174180":{"success":true,"data":{"type":"game","name":"Red Dead Redemption 2",
		"header_image":"https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/1174180/header.jpg?t=1759502961"}}}`

	got, err := decodeStore(strings.NewReader(ok), "1174180")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got == nil || got.Name != "Red Dead Redemption 2" || got.Type != "game" {
		t.Fatalf("got %+v", got)
	}
	if !strings.HasSuffix(got.HeaderImage, "header.jpg?t=1759502961") {
		t.Errorf("header = %q", got.HeaderImage)
	}
}

func TestDecodeStoreMissingApp(t *testing.T) {
	// What a runtime appid actually returns
	got, err := decodeStore(strings.NewReader(`{"1070560":{"success":false}}`), "1070560")
	if err != nil || got != nil {
		t.Errorf("got %+v err=%v, want a cacheable nothing", got, err)
	}
}

func TestDecodeStoreJunk(t *testing.T) {
	if _, err := decodeStore(strings.NewReader(`<html>rate limited</html>`), "1"); err == nil {
		t.Error("html decoded as a store answer")
	}
}

func TestStaleBackoff(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	r := &resolver{now: func() time.Time { return now }}

	cases := []struct {
		name string
		a    art
		want bool
	}{
		{"never asked", art{}, true},
		{"resolved yesterday stays resolved", art{Store: storeOK, ResolvedAt: now.Add(-24 * time.Hour)}, false},
		{"an attempt just made is not repeated", art{Attempts: 1, ResolvedAt: now.Add(-30 * time.Second)}, false},
		{"first error retries after two minutes", art{Store: storeErrored, Attempts: 1, ResolvedAt: now.Add(-3 * time.Minute)}, true},
		{"sixth error waits an hour", art{Store: storeErrored, Attempts: 6, ResolvedAt: now.Add(-40 * time.Minute)}, false},
		{"backoff is capped", art{Store: storeErrored, Attempts: 40, ResolvedAt: now.Add(-3 * time.Hour)}, true},
		{"missing is rechecked monthly", art{Store: storeMissing, Attempts: 1, ResolvedAt: now.Add(-10 * 24 * time.Hour)}, false},
		{"missing goes stale eventually", art{Store: storeMissing, Attempts: 1, ResolvedAt: now.Add(-40 * 24 * time.Hour)}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := r.stale(c.a); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestArtCacheRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	r := newResolver("")

	want := art{V: artVersion, AppID: 1174180, Name: "Red Dead Redemption 2",
		Store: storeOK, HeroURL: "https://example.invalid/hero.jpg"}
	r.write(want)

	got, err := r.read(1174180)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Name != want.Name || got.HeroURL != want.HeroURL || got.Store != storeOK {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestArtCacheRejectsBadFiles(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	r := newResolver("")

	if _, err := r.read(1); err == nil {
		t.Error("read a file that does not exist")
	}

	r.write(art{V: artVersion + 1, AppID: 2, Store: storeOK})
	if _, err := r.read(2); err == nil {
		t.Error("a version we do not understand read as usable")
	}

	r.write(art{V: artVersion, AppID: 3, Store: storeOK})
	if _, err := r.read(4); err == nil {
		t.Error("read someone else's appid")
	}
}
