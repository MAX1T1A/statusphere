package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	nudgeFrom = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	nudgeText = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	nudgeTime = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

const nudgeMaxHistory = 5

type NudgeEntry struct {
	From    string
	Message string
	At      time.Time
}

type NudgeHistory struct {
	mu      sync.Mutex
	entries []NudgeEntry
	localID string
	seen    map[string]string
}

func NewNudgeHistory(localID string) *NudgeHistory {
	return &NudgeHistory{
		localID: localID,
		seen:    make(map[string]string),
	}
}

func (h *NudgeHistory) Process(deviceID, deviceName, message string) {
	if deviceID == h.localID || message == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.seen[deviceID] == message {
		return
	}
	h.seen[deviceID] = message

	h.entries = append(h.entries, NudgeEntry{
		From:    deviceName,
		Message: message,
		At:      time.Now(),
	})
	if len(h.entries) > nudgeMaxHistory {
		h.entries = h.entries[len(h.entries)-nudgeMaxHistory:]
	}
}

func (h *NudgeHistory) Render() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 {
		return ""
	}

	var lines []string
	for _, e := range h.entries {
		ago := time.Since(e.At)
		var ts string
		if ago < time.Minute {
			ts = "now"
		} else if ago < time.Hour {
			ts = fmt.Sprintf("%dm", int(ago.Minutes()))
		} else {
			ts = fmt.Sprintf("%dh", int(ago.Hours()))
		}

		lines = append(lines, fmt.Sprintf("%s %s %s",
			nudgeFrom.Render(e.From+":"),
			nudgeText.Render(e.Message),
			nudgeTime.Render("· "+ts),
		))
	}

	return strings.Join(lines, "\n")
}

func BlockNudge(history *NudgeHistory) Block {
	return Block{
		Key: "nudge",
		Render: func(d map[string]any) string {
			return ""
		},
	}
}
