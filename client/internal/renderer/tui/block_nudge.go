package tui

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	nudgeMsg    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	nudgeSelf   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	nudgeTime   = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
	nudgeSelfLb = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

const nudgeMax = 13

type NudgeEntry struct {
	Message string
	At      time.Time
	Self    bool
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

	h.push(deviceID, message, false)
}

func (h *NudgeHistory) ProcessLocal(message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.push("__self__", message, true)
}

func (h *NudgeHistory) push(deviceID, message string, self bool) {
	entries := append(h.devices[deviceID], NudgeEntry{
		Message: message,
		At:      time.Now(),
		Self:    self,
	})
	if len(entries) > nudgeMax {
		entries = entries[len(entries)-nudgeMax:]
	}
	h.devices[deviceID] = entries
}

func (h *NudgeHistory) RenderFor(deviceID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	remote := h.devices[deviceID]
	self := h.devices["__self__"]

	if len(remote) == 0 && len(self) == 0 {
		return ""
	}

	merged := make([]NudgeEntry, 0, len(remote)+len(self))
	merged = append(merged, remote...)
	merged = append(merged, self...)

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].At.Before(merged[j].At)
	})

	if len(merged) > nudgeMax {
		merged = merged[len(merged)-nudgeMax:]
	}

	var lines []string
	for _, e := range merged {
		ts := e.At.Format("15:04")

		if e.Self {
			lines = append(lines, nudgeSelf.Render(e.Message)+" "+nudgeTime.Render("· "+ts))
		} else {
			lines = append(lines, nudgeMsg.Render(e.Message)+" "+nudgeTime.Render("· "+ts))
		}
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
