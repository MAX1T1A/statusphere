package stats_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"statusphere-client/internal/stats"
)

func TestGetSyncFetchesAndParses(t *testing.T) {
	var gotPath, gotToken, gotDevice, gotRoom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Room-Token")
		gotDevice = r.URL.Query().Get("device_id")
		gotRoom = r.URL.Query().Get("room")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_id":"dev1","period":"week","total_seconds":3600,"daily":[{"day":"2026-07-20","seconds":1800}]}`))
	}))
	defer srv.Close()

	cache := stats.NewSpotifyCache(srv.URL, "tok123", "room9")
	result := cache.GetSync("dev1")

	s, ok := result.(*stats.SpotifyStats)
	if !ok || s == nil {
		t.Fatalf("expected *SpotifyStats, got %T", result)
	}
	if s.TotalSeconds != 3600 || len(s.Daily) != 1 || s.Daily[0].Seconds != 1800 {
		t.Fatalf("unexpected parse: %+v", s)
	}
	if gotPath != "/stats/spotify" || gotToken != "tok123" || gotDevice != "dev1" || gotRoom != "room9" {
		t.Fatalf("request mismatch: path=%s token=%s device=%s room=%s", gotPath, gotToken, gotDevice, gotRoom)
	}
}

func TestGetEmptyDeviceReturnsNil(t *testing.T) {
	cache := stats.NewSummaryCache("http://example.invalid", "tok", "day", "room9")
	if cache.Get("") != nil {
		t.Fatal("empty device id should return nil")
	}
}
