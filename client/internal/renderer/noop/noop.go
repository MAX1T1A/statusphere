package noop

import "statusphere-client/internal/presence"

type Noop struct {
	done chan struct{}
}

func NewNoop() *Noop {
	return &Noop{done: make(chan struct{})}
}

func (n *Noop) Run() error {
	<-n.done
	return nil
}

func (n *Noop) Stop() {
	close(n.done)
}

func (n *Noop) UpdateDevices([]presence.Snapshot) {}
