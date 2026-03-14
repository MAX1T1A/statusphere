package notifier

import (
	"os/exec"
	"sync"
)

type Notifier struct {
	localID string
	mu      sync.Mutex
	last    map[string]string
}

func New(localDeviceID string) *Notifier {
	return &Notifier{
		localID: localDeviceID,
		last:    make(map[string]string),
	}
}

func (n *Notifier) Handle(deviceID, deviceName, message string) {
	if deviceID == n.localID || message == "" {
		return
	}

	n.mu.Lock()
	if n.last[deviceID] == message {
		n.mu.Unlock()
		return
	}
	n.last[deviceID] = message
	n.mu.Unlock()

	exec.Command("notify-send", "statusphere · "+deviceName, message).Start()
}
