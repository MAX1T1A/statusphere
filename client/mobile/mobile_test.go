package mobile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/websocket"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/cardlayout"
	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
	"statusphere-client/internal/privacy"
)

const waitFrame = 3 * time.Second

type fakeConn struct {
	ws     *websocket.Conn
	frames chan map[string]any
}

type fakeServer struct {
	*httptest.Server
	conns   chan *fakeConn
	members []auth.MemberInfo
}

func newFakeServer(t *testing.T, members ...auth.MemberInfo) *fakeServer {
	srv := &fakeServer{conns: make(chan *fakeConn, 4), members: members}
	mux := http.NewServeMux()
	mux.HandleFunc("/rooms/members", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"members": srv.members})
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		fc := &fakeConn{ws: ws, frames: make(chan map[string]any, 16)}
		srv.conns <- fc
		defer close(fc.frames)
		for {
			_, data, err := ws.Read(r.Context())
			if err != nil {
				return
			}
			var frame map[string]any
			if json.Unmarshal(data, &frame) == nil {
				fc.frames <- frame
			}
		}
	})
	srv.Server = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func (srv *fakeServer) nextConn(t *testing.T, within time.Duration) *fakeConn {
	t.Helper()
	select {
	case fc := <-srv.conns:
		return fc
	case <-time.After(within):
		t.Fatal("client did not connect")
		return nil
	}
}

func (fc *fakeConn) next(t *testing.T) map[string]any {
	t.Helper()
	select {
	case f, ok := <-fc.frames:
		if !ok {
			t.Fatal("connection closed before the expected frame")
		}
		return f
	case <-time.After(waitFrame):
		t.Fatal("no frame from the client")
		return nil
	}
}

func (fc *fakeConn) none(t *testing.T, within time.Duration) {
	t.Helper()
	select {
	case f := <-fc.frames:
		t.Fatalf("expected no frame, got %v", f)
	case <-time.After(within):
	}
}

func (fc *fakeConn) push(t *testing.T, snap presence.Snapshot) {
	t.Helper()
	data, _ := json.Marshal(snap)
	if err := fc.ws.Write(context.Background(), websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}

type recorder struct {
	rooms  chan string
	errors chan string
}

func newRecorder() *recorder {
	return &recorder{rooms: make(chan string, 64), errors: make(chan string, 64)}
}

func (r *recorder) OnRoom(roomJSON string)              { r.rooms <- roomJSON }
func (r *recorder) OnError(event string, detail string) { r.errors <- event + ": " + detail }

func baseDir(t *testing.T, serverURL string) string {
	t.Helper()
	dir := t.TempDir()
	config.SetDir(dir)
	t.Cleanup(func() { config.SetDir("") })
	cfg := &auth.Config{
		ServerURL: serverURL,
		AccountID: "acc-me",
		DeviceID:  "phone-1",
		Token:     "tok",
		RoomID:    "room-1",
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func startSession(t *testing.T, srv *fakeServer, dir string) (*Session, *fakeConn, *recorder) {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	rec := newRecorder()
	if err := s.Start(rec); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s, srv.nextConn(t, waitFrame), rec
}

const (
	trackA = `{"music":{"track":"Roygbiv","artist":"Boards of Canada","status":"Playing","position_seconds":12,"length_seconds":150}}`
	trackB = `{"music":{"track":"Xtal","artist":"Aphex Twin","status":"Playing","position_seconds":3,"length_seconds":290}}`
)

func TestPublishSendsOnlyOnChange(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))

	if err := s.Publish(trackA); err != nil {
		t.Fatal(err)
	}
	first := conn.next(t)
	if first[presence.KeySpotifyTrack] != "Roygbiv" || first[presence.KeySpotifyStatus] != "playing" {
		t.Fatalf("first publish should carry the track, got %v", first)
	}
	if first[presence.KeyDeviceID] != "phone-1" {
		t.Fatalf("frame should carry this device id, got %v", first)
	}

	if err := s.Publish(trackA); err != nil {
		t.Fatal(err)
	}
	conn.none(t, 300*time.Millisecond)

	if err := s.Publish(trackB); err != nil {
		t.Fatal(err)
	}
	if got := conn.next(t); got[presence.KeySpotifyTrack] != "Xtal" {
		t.Fatalf("a changed track should be sent, got %v", got)
	}
}

func TestPublishRejectsMalformedSnapshot(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))

	if err := s.Publish("{not json"); err == nil {
		t.Fatal("malformed snapshot json should be an error")
	}
	conn.none(t, 200*time.Millisecond)
}

func TestHeartbeatResendsUnchangedState(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))
	if err := s.SetHeartbeatSeconds(1); err != nil {
		t.Fatal(err)
	}

	if err := s.Publish(trackA); err != nil {
		t.Fatal(err)
	}
	conn.next(t)

	if got := conn.next(t); got[presence.KeySpotifyTrack] != "Roygbiv" {
		t.Fatalf("heartbeat should resend the current state, got %v", got)
	}
}

func TestHeartbeatMustStayBelowStaleTTL(t *testing.T) {
	srv := newFakeServer(t)
	s, _, _ := startSession(t, srv, baseDir(t, srv.URL))

	for _, seconds := range []int{0, -1, 300, 600} {
		if err := s.SetHeartbeatSeconds(seconds); err == nil {
			t.Fatalf("heartbeat of %ds should be rejected", seconds)
		}
	}
	if err := s.SetHeartbeatSeconds(180); err != nil {
		t.Fatalf("a 3 min heartbeat is what a screen-off phone uses: %v", err)
	}
}

func TestListenStateAndSnapshotResentAfterReconnect(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))

	s.SetListening(false)
	if got := conn.next(t); got["type"] != "listen" || got["on"] != false {
		t.Fatalf("turning listening off should send a listen frame, got %v", got)
	}
	if err := s.Publish(trackA); err != nil {
		t.Fatal(err)
	}
	conn.next(t)

	_ = conn.ws.Close(websocket.StatusGoingAway, "restart")
	again := srv.nextConn(t, 10*time.Second)

	if got := again.next(t); got["type"] != "listen" || got["on"] != false {
		t.Fatalf("a new connection listens by default, so listen off must be resent first, got %v", got)
	}
	if got := again.next(t); got[presence.KeySpotifyTrack] != "Roygbiv" {
		t.Fatalf("the current snapshot should be resent after reconnect, got %v", got)
	}
}

func TestListeningOnSendsListenFrame(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))

	s.SetListening(true)
	conn.none(t, 200*time.Millisecond)

	s.SetListening(false)
	conn.next(t)
	s.SetListening(true)
	if got := conn.next(t); got["type"] != "listen" || got["on"] != true {
		t.Fatalf("resuming should send listen on, got %v", got)
	}
}

func writePrivacy(t *testing.T, dir string, p privacy.Policy) {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, privacy.FileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPrivacyFilterRunsBeforeSend(t *testing.T) {
	srv := newFakeServer(t)
	dir := baseDir(t, srv.URL)
	p := privacy.Default()
	p.Mode = privacy.ModeIncognito
	writePrivacy(t, dir, p)
	s, conn, _ := startSession(t, srv, dir)

	err := s.Publish(`{"music":{"track":"Roygbiv","artist":"Boards of Canada","status":"playing"},"app":{"label":"Telegram","package":"org.telegram.messenger"}}`)
	if err != nil {
		t.Fatal(err)
	}
	got := conn.next(t)
	if _, ok := got[presence.KeyActiveApp]; ok {
		t.Fatalf("incognito hides apps, got %v", got)
	}
	if got[presence.KeySpotifyTrack] != "Roygbiv" {
		t.Fatalf("default incognito keeps music, got %v", got)
	}
	if got[presence.KeyIncognito] != true {
		t.Fatalf("an announced incognito should mark the frame, got %v", got)
	}
}

func TestHideAppsMatchesPackageAndPackageNeverLeaves(t *testing.T) {
	srv := newFakeServer(t)
	s, conn, _ := startSession(t, srv, baseDir(t, srv.URL))

	if err := s.Publish(`{"app":{"label":"Vault","package":"com.x8bit.bitwarden"}}`); err != nil {
		t.Fatal(err)
	}
	if got := conn.next(t); got[presence.KeyActiveApp] != nil {
		t.Fatalf("a hide_apps pattern matching the package should hide the app, got %v", got)
	}

	if err := s.Publish(`{"app":{"label":"Telegram","package":"org.telegram.messenger"}}`); err != nil {
		t.Fatal(err)
	}
	got := conn.next(t)
	if got[presence.KeyActiveApp] != "Telegram" {
		t.Fatalf("a visible app should go out by its label, got %v", got)
	}
	for k, v := range got {
		if v == "org.telegram.messenger" {
			t.Fatalf("the package is only for matching, yet it went out as %q", k)
		}
	}
}

type roomPayload struct {
	Members []presence.Snapshot `json:"members"`
	Photos  []json.RawMessage   `json:"photos"`
	Cards   []cardlayout.Card   `json:"cards"`
}

func TestRoomJSONMergesLiveDevicesWithMembers(t *testing.T) {
	srv := newFakeServer(t,
		auth.MemberInfo{AccountID: "acc-bob", Name: "Bob", Role: "owner"},
		auth.MemberInfo{AccountID: "acc-ann", Name: "Ann", Role: "member"},
	)
	_, conn, rec := startSession(t, srv, baseDir(t, srv.URL))

	conn.push(t, presence.Snapshot{
		presence.KeyDeviceID:     "bob-1",
		presence.KeyAccountID:    "acc-bob",
		presence.KeySpotifyTrack: "Xtal",
	})

	deadline := time.After(waitFrame)
	for {
		var raw string
		select {
		case raw = <-rec.rooms:
		case <-deadline:
			t.Fatal("no room update with bob online and ann offline")
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &fields); err != nil {
			t.Fatalf("room is not a json object: %v", err)
		}
		if len(fields) != 3 || fields["members"] == nil || string(fields["photos"]) != "[]" || fields["cards"] == nil {
			t.Fatalf("room shape should be {members, photos: [], cards}, got %s", raw)
		}

		var room roomPayload
		if err := json.Unmarshal([]byte(raw), &room); err != nil {
			t.Fatal(err)
		}
		byAcc := map[string]presence.Snapshot{}
		for _, m := range room.Members {
			byAcc[m.String(presence.KeyAccountID)] = m
		}
		bob, ann := byAcc["acc-bob"], byAcc["acc-ann"]
		if bob == nil || bob.Has(presence.KeyOffline) || ann == nil {
			continue
		}
		if bob.String(presence.KeyRole) != "owner" || bob.String(presence.KeySpotifyTrack) != "Xtal" {
			t.Fatalf("bob should be live with his role and track, got %v", bob)
		}
		if !ann.Has(presence.KeyOffline) || ann.String(presence.KeyAccountName) != "Ann" {
			t.Fatalf("ann should be an offline placeholder, got %v", ann)
		}
		cardAccounts := map[string]bool{}
		for _, c := range room.Cards {
			cardAccounts[c.AccountID] = true
		}
		if len(room.Cards) != 2 || !cardAccounts["acc-bob"] || !cardAccounts["acc-ann"] {
			t.Fatalf("room should carry one card per account, got %+v", room.Cards)
		}
		return
	}
}

func TestJoinRegistersAndPublishesUnderThePhoneDeviceName(t *testing.T) {
	var registeredAs string
	mux := http.NewServeMux()
	mux.HandleFunc("/accounts/register", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		registeredAs = body["name"]
		_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acc", "device_id": "dev", "room_id": "own", "token": "tok"})
	})
	mux.HandleFunc("/rooms/join", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"room_id": "room"})
	})
	mux.HandleFunc("/accounts/name", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { config.SetDir("") })

	if err := Join(t.TempDir(), auth.EncodeInvite(srv.URL, "code"), "Ann"); err != nil {
		t.Fatalf("join: %v", err)
	}
	if registeredAs != phoneDeviceName {
		t.Fatalf("registered as %q, want %q", registeredAs, phoneDeviceName)
	}
	if got := config.DeviceName(); got != phoneDeviceName {
		t.Fatalf("published device name %q, want %q", got, phoneDeviceName)
	}
}
