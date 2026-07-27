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

type payload struct {
	Members []presence.Snapshot `json:"members"`
}

type JSONLine struct {
	out  io.Writer
	done chan struct{}

	mu   sync.Mutex
	last []byte
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
	data, err := json.Marshal(payload{Members: devices})
	if err != nil {
		return
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	if bytes.Equal(data, j.last) {
		return
	}
	j.last = data
	data = append(data, '\n')
	_, _ = j.out.Write(data)
}
