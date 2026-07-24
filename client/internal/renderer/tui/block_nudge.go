package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	chatTime = lipgloss.NewStyle().Foreground(cDim)
	chatMsg  = lipgloss.NewStyle().Foreground(cValue)
	chatYou  = lipgloss.NewStyle().Bold(true).Foreground(cYou)
	chatName = lipgloss.NewStyle().Bold(true).Foreground(cOther)
)

const chatMax = 200

type NudgeEntry struct {
	Sender  string
	Name    string
	Message string
	At      time.Time
	Self    bool
}

type NudgeHistory struct {
	mu      sync.Mutex
	log     []NudgeEntry
	localID string
	seen    map[string]string
}

func NewNudgeHistory(localID string) *NudgeHistory {
	return &NudgeHistory{
		localID: localID,
		seen:    make(map[string]string),
	}
}

func (h *NudgeHistory) append(e NudgeEntry) {
	h.log = append(h.log, e)
	if len(h.log) > chatMax {
		h.log = h.log[len(h.log)-chatMax:]
	}
}

func (h *NudgeHistory) Process(sender, name, message string) {
	if sender == h.localID || message == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.seen[sender] == message {
		return
	}
	h.seen[sender] = message
	h.append(NudgeEntry{Sender: sender, Name: name, Message: message, At: time.Now()})
}

func (h *NudgeHistory) ProcessLocal(message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.append(NudgeEntry{Message: message, At: time.Now(), Self: true})
}

func (h *NudgeHistory) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.log)
}

func (h *NudgeHistory) Render() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.log) == 0 {
		return dimStyle.Render("no messages yet")
	}

	lines := make([]string, 0, len(h.log))
	for _, e := range h.log {
		ts := chatTime.Render(e.At.Format("15:04"))
		var who string
		if e.Self {
			who = chatYou.Render("you")
		} else if e.Name != "" {
			who = chatName.Render(e.Name)
		} else {
			who = chatName.Render("someone")
		}
		lines = append(lines, ts+"  "+who+chatTime.Render(": ")+chatMsg.Render(e.Message))
	}
	return strings.Join(lines, "\n")
}
