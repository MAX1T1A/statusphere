package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	nudgeMsg  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	nudgeTime = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

const nudgeMax = 5

type NudgeEntry struct {
	Message string
	At      time.Time
}

type NudgeHistory struct {
	mu      sync.Mutex
	devices map[string][]NudgeEntry
	localID string
	seen    map[string]string
}

func NewNudgeHistory(localID string) *NudgeHistory {
	return &NudgeHistory{
		localID: localID,
		devices: make(map[string][]NudgeEntry),
		seen:    make(map[string]string),
	}
}

func (h *NudgeHistory) Process(deviceID, message string) {
	if deviceID == h.localID || message == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.seen[deviceID] == message {
		return
	}
	h.seen[deviceID] = message

	entries := append(h.devices[deviceID], NudgeEntry{
		Message: message,
		At:      time.Now(),
	})
	if len(entries) > nudgeMax {
		entries = entries[len(entries)-nudgeMax:]
	}
	h.devices[deviceID] = entries
}

func (h *NudgeHistory) RenderFor(deviceID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries := h.devices[deviceID]
	if len(entries) == 0 {
		return ""
	}

	var lines []string
	for _, e := range entries {
		ago := time.Since(e.At)
		var ts string
		if ago < time.Minute {
			ts = "now"
		} else if ago < time.Hour {
			ts = fmt.Sprintf("%dm", int(ago.Minutes()))
		} else {
			ts = fmt.Sprintf("%dh", int(ago.Hours()))
		}

		lines = append(lines, nudgeMsg.Render(e.Message)+" "+nudgeTime.Render("· "+ts))
	}

	return strings.Join(lines, "\n")
}

func BlockNudge(history *NudgeHistory) Block {
	return Block{
		Key: "nudge",
		Render: func(d map[string]any) string {
			deviceID, _ := d["device_id"].(string)
			if deviceID == "" || history == nil {
				return ""
			}
			return history.RenderFor(deviceID)
		},
	}
}
