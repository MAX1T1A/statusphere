package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"statusphere-client/internal/auth"
)

func TestRoomScopedMethodsRequireRoom(t *testing.T) {
	// No server is started: a request would fail to connect, so reaching
	// ErrNoRoom instead proves the guard runs before any network call.
	c := &auth.Config{ServerURL: "http://127.0.0.1:1", Token: "t"}

	cases := []struct {
		name string
		call func() error
	}{
		{"Invite", func() error { _, err := c.Invite(); return err }},
		{"Members", func() error { _, err := c.Members(); return err }},
		{"Kick", func() error { _, err := c.Kick("acc"); return err }},
		{"Leave", func() error { _, err := c.Leave(); return err }},
		{"Promote", func() error { _, err := c.Promote("acc"); return err }},
		{"Demote", func() error { _, err := c.Demote("acc"); return err }},
		{"PostPhoto", func() error { _, err := c.PostPhoto("/nonexistent"); return err }},
		{"ListRoomPhotos", func() error { _, err := c.ListRoomPhotos(); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if !errors.Is(err, auth.ErrNoRoom) {
				t.Fatalf("got %v, want ErrNoRoom", err)
			}
		})
	}
}

func TestDoSurfacesServerErrorDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "not a room member"})
	}))
	t.Cleanup(srv.Close)

	c := &auth.Config{ServerURL: srv.URL, Token: "t", RoomID: "room1"}
	_, err := c.Members()
	if err == nil {
		t.Fatal("want error")
	}
	if got := err.Error(); !strings.Contains(got, "not a room member") {
		t.Fatalf("error %q does not surface the server detail", got)
	}
}
