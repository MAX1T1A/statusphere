// Package jsonline renders the room roster as one JSON object per line on stdout,
// for consumption by external UIs (e.g. a status bar widget) that spawn this binary.
package jsonline

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"statusphere-client/internal/presence"
)

// PhotoOut is a room member's current shared photo, wire shape for the "json"
// UI mode. Path is a local file:// URI the CLI has already fetched and cached -
// Quickshell's Image element has no way to attach the auth header the blob
// endpoint requires, so the CLI does that fetch on its behalf.
type PhotoOut struct {
	AccountID string `json:"account_id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type payload struct {
	Members []presence.Snapshot `json:"members"`
	Photos  []PhotoOut          `json:"photos"`
}

type JSONLine struct {
	out  io.Writer
	done chan struct{}

	mu      sync.Mutex
	last    []byte
	members []presence.Snapshot
	photos  []PhotoOut
}

func New(out io.Writer) *JSONLine {
	return &JSONLine{out: out, done: make(chan struct{})}
}

func (j *JSONLine) Run() error {
	<-j.done
	return nil
}

func (j *JSONLine) Stop() {
	close(j.done)
}

func (j *JSONLine) UpdateDevices(devices []presence.Snapshot) {
	j.mu.Lock()
	j.members = devices
	data, err := json.Marshal(payload{Members: j.members, Photos: j.photos})
	j.mu.Unlock()
	j.emit(data, err)
}

func (j *JSONLine) UpdatePhotos(photos []PhotoOut) {
	j.mu.Lock()
	j.photos = photos
	data, err := json.Marshal(payload{Members: j.members, Photos: j.photos})
	j.mu.Unlock()
	j.emit(data, err)
}

func (j *JSONLine) emit(data []byte, err error) {
	if err != nil {
		return
	}

	j.mu.Lock()
	if bytes.Equal(data, j.last) {
		j.mu.Unlock()
		return
	}
	j.last = data
	j.mu.Unlock()

	data = append(data, '\n')
	_, _ = j.out.Write(data)
}
