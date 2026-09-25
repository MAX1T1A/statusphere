// Package mobile is the surface gomobile binds for the Android app. Exported
// signatures stay within the types gomobile can bind: string, bool, int,
// float64, []byte, error and this package's own structs and interfaces.
package mobile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"statusphere-client/internal/auth"
	"statusphere-client/internal/config"
	"statusphere-client/internal/feed"
	"statusphere-client/internal/layout"
	"statusphere-client/internal/presence"
	"statusphere-client/internal/privacy"
	"statusphere-client/internal/renderer/jsonline"
	"statusphere-client/internal/transport"
	"statusphere-client/internal/watcher"
)

type RoomListener interface {
	OnRoom(roomJSON string)
	// OnError reports a failure; an empty event clears the previous one.
	OnError(event string, detail string)
}

var errStarted = errors.New("session already started")

const phoneDeviceName = "phone"

func Join(baseDir, invite, name string) error {
	config.SetDir(baseDir)
	if err := config.SetDeviceName(phoneDeviceName); err != nil {
		return err
	}
	cfg, _, err := auth.JoinInvite(invite)
	if err != nil {
		return err
	}
	if name == "" {
		return nil
	}
	return cfg.SetAccountName(name)
}

type Session struct {
	cfg     *auth.Config
	ws      *transport.WSTransport
	feed    *feed.Feed
	roster  *feed.Roster
	privacy *privacy.Store
	layout  *layout.Store
	rearm   chan struct{}
	dirty   chan struct{}

	mu         sync.Mutex
	listener   RoomListener
	ctx        context.Context
	cancel     context.CancelFunc
	pollCancel context.CancelFunc
	listening  bool
	heartbeat  time.Duration
	lastRoom   []byte

	publishMu sync.Mutex
	current   presence.Snapshot
	gate      watcher.Gate
}

func Open(baseDir string) (*Session, error) {
	config.SetDir(baseDir)
	cfg, err := auth.Load()
	if err != nil {
		return nil, err
	}
	if cfg.RoomID == "" {
		return nil, auth.ErrNoRoom
	}
	privacy.EnsureConfig()

	s := &Session{
		cfg:       cfg,
		ws:        transport.NewWS(cfg.ServerURL, cfg.Token, cfg.DeviceID, cfg.RoomID),
		feed:      feed.New(),
		roster:    feed.NewRoster(cfg.Members),
		privacy:   &privacy.Store{},
		layout:    &layout.Store{},
		rearm:     make(chan struct{}, 1),
		dirty:     make(chan struct{}, 1),
		listening: true,
		heartbeat: watcher.DefaultHeartbeat,
	}
	s.ws.OnConnect(s.restore)
	return s, nil
}

func (s *Session) Start(listener RoomListener) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return errStarted
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.listener = listener
	s.lastRoom = nil
	if s.listening {
		s.startPollLocked()
	}
	ctx := s.ctx
	s.mu.Unlock()

	if err := s.ws.Connect(ctx); err != nil {
		listener.OnError("ws_connect_failed", err.Error())
	}
	go s.ws.Listen(ctx, s.receive)
	go s.beat(ctx)
	go s.deliverRooms(ctx)
	return nil
}

func (s *Session) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel, s.ctx, s.pollCancel, s.listener = nil, nil, nil, nil
	s.mu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	_ = s.ws.Close()
}

func (s *Session) Leave() (bool, error) {
	ok, err := s.cfg.Leave()
	if err != nil {
		return false, err
	}
	if ok {
		s.Stop()
	}
	return ok, nil
}

func (s *Session) SetName(name string) error {
	return s.cfg.SetAccountName(name)
}

func (s *Session) Publish(snapshotJSON string) error {
	snap, err := parsePhoneSnapshot(snapshotJSON)
	if err != nil {
		return err
	}
	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	s.current = snap
	s.offerLocked(s.currentHeartbeat())
	return nil
}

func (s *Session) SetListening(on bool) {
	s.mu.Lock()
	if s.listening == on {
		s.mu.Unlock()
		return
	}
	s.listening = on
	if on {
		s.lastRoom = nil
		s.startPollLocked()
	} else {
		s.stopPollLocked()
	}
	s.mu.Unlock()

	if err := s.ws.SendListen(on); err != nil {
		log.Printf("mobile_listen_send_failed on=%t err=%q", on, err)
	}
	if on {
		s.emit()
	}
}

func (s *Session) SetHeartbeatSeconds(seconds int) error {
	d := time.Duration(seconds) * time.Second
	if d <= 0 || d >= feed.StaleTTL {
		return fmt.Errorf("heartbeat must be positive and below %s, got %s", feed.StaleTTL, d)
	}
	s.mu.Lock()
	s.heartbeat = d
	s.mu.Unlock()
	select {
	case s.rearm <- struct{}{}:
	default:
	}
	return nil
}

func (s *Session) SetPingSeconds(seconds int) error {
	if seconds <= 0 {
		return fmt.Errorf("ping interval must be positive, got %ds", seconds)
	}
	s.ws.SetPingInterval(time.Duration(seconds) * time.Second)
	return nil
}

func (s *Session) AccountID() string { return s.cfg.AccountID }

// NetworkAvailable nudges an immediate roster refresh and drops the current
// WS connection, for callers that observe the phone's connectivity changed
// (e.g. wifi to mobile data) instead of waiting for a poll tick or a stale
// read to notice.
func (s *Session) NetworkAvailable() {
	s.roster.Kick()
	s.ws.Kick()
}

const IncognitoUntilTurnedOff = 0

func (s *Session) SetIncognito(on bool, minutes int) error {
	mode := privacy.ModeNormal
	if on {
		mode = privacy.ModeIncognito
	}
	if _, err := privacy.Set(mode, time.Duration(minutes)*time.Minute); err != nil {
		return err
	}

	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	s.privacy = &privacy.Store{}
	s.offerLocked(s.currentHeartbeat())
	return nil
}

func (s *Session) Incognito() bool { return s.policy().Hidden() }

func (s *Session) IncognitoUntilUnix() int64 {
	until, ok := s.policy().Expires()
	if !ok {
		return 0
	}
	return until.Unix()
}

func (s *Session) policy() privacy.Policy {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	return s.privacy.Policy()
}

func (s *Session) currentHeartbeat() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.heartbeat
}

func (s *Session) offerLocked(heartbeat time.Duration) {
	if s.current == nil {
		return
	}
	out := s.layout.Annotate(withoutPackage(s.privacy.Apply(s.current)))
	if !s.gate.Pass(out, heartbeat) {
		return
	}
	if err := s.ws.Send(out); err != nil {
		log.Printf("mobile_presence_send_failed err=%q", err)
	}
	s.feed.UpdateOwn(out, s.cfg.DeviceID, s.cfg.AccountID, s.ws.DeviceName())
	s.emit()
}

const alwaysDue = 0

func (s *Session) restore() {
	s.mu.Lock()
	listening := s.listening
	s.mu.Unlock()

	if !listening {
		if err := s.ws.SendListen(false); err != nil {
			log.Printf("mobile_listen_send_failed on=false err=%q", err)
		}
	}

	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	s.offerLocked(alwaysDue)
}

func (s *Session) beat(ctx context.Context) {
	for {
		heartbeat := s.currentHeartbeat()
		s.publishMu.Lock()
		wait := time.Until(s.gate.NextHeartbeat(heartbeat))
		s.publishMu.Unlock()
		if wait <= 0 {
			wait = heartbeat
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.rearm:
			timer.Stop()
		case <-timer.C:
			s.publishMu.Lock()
			s.offerLocked(s.currentHeartbeat())
			s.publishMu.Unlock()
		}
	}
}

func (s *Session) startPollLocked() {
	if s.ctx == nil || s.pollCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.pollCancel = cancel
	go s.roster.Poll(ctx, s.membersRefreshed)
}

func (s *Session) stopPollLocked() {
	if s.pollCancel != nil {
		s.pollCancel()
		s.pollCancel = nil
	}
}

func (s *Session) membersRefreshed(err error) {
	if err != nil {
		s.fail("room_members_fetch_failed", err)
		return
	}
	s.clearError()
	s.emit()
}

func (s *Session) receive(data []byte) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	switch msg["type"] {
	case "msg", "photo_status":
		return
	}
	snap := presence.Snapshot(msg)
	s.feed.Update(snap)
	s.roster.Seen(snap.String(presence.KeyAccountID))
	s.emit()
}

func (s *Session) emit() {
	select {
	case s.dirty <- struct{}{}:
	default:
	}
}

func (s *Session) deliverRooms(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.dirty:
			s.deliverRoom()
		}
	}
}

func (s *Session) deliverRoom() {
	room, err := jsonline.Encode(s.roster.Merge(s.feed.Snapshot()), []jsonline.PhotoOut{})
	if err != nil {
		s.fail("room_encode_failed", err)
		return
	}

	s.mu.Lock()
	listener := s.listener
	if listener == nil || !s.listening || string(room) == string(s.lastRoom) {
		s.mu.Unlock()
		return
	}
	s.lastRoom = room
	s.mu.Unlock()

	listener.OnRoom(string(room))
}

func (s *Session) fail(event string, err error) {
	s.mu.Lock()
	listener := s.listener
	s.mu.Unlock()
	if listener != nil {
		listener.OnError(event, err.Error())
	}
}

func (s *Session) clearError() {
	s.mu.Lock()
	listener := s.listener
	s.mu.Unlock()
	if listener != nil {
		listener.OnError("", "")
	}
}
