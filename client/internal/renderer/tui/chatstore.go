package tui

import (
	"sort"
	"sync"
	"time"

	"statusphere-client/internal/chat"

	"github.com/charmbracelet/lipgloss"
)

var (
	chatTime = lipgloss.NewStyle().Foreground(cDim)
	chatMsg  = lipgloss.NewStyle().Foreground(cValue)
	chatYou  = lipgloss.NewStyle().Bold(true).Foreground(cYou)
	chatName = lipgloss.NewStyle().Bold(true).Foreground(cOther)
)

const (
	chatMax     = 300
	groupThread = ""
)

type ChatEntry struct {
	From  string
	Name  string
	Text  string
	At    time.Time
	Self  bool
	atKey string
}

type thread struct {
	log    []ChatEntry
	unread int
}

func (t *thread) contains(e ChatEntry) bool {
	if e.atKey == "" {
		return false
	}
	for i := len(t.log) - 1; i >= 0; i-- {
		x := t.log[i]
		if x.atKey == e.atKey && x.From == e.From && x.Text == e.Text {
			return true
		}
	}
	return false
}

func (t *thread) add(e ChatEntry) bool {
	if t.contains(e) {
		return false
	}
	t.log = append(t.log, e)
	if len(t.log) > chatMax {
		t.log = t.log[len(t.log)-chatMax:]
	}
	return true
}

func (t *thread) sortAndCap() {
	sort.SliceStable(t.log, func(i, j int) bool { return t.log[i].At.Before(t.log[j].At) })
	if len(t.log) > chatMax {
		t.log = t.log[len(t.log)-chatMax:]
	}
}

type ChatStore struct {
	mu      sync.Mutex
	localID string
	group   thread
	dms     map[string]*thread
}

func NewChatStore(localAccountID string) *ChatStore {
	return &ChatStore{localID: localAccountID, dms: make(map[string]*thread)}
}

func (c *ChatStore) threadFor(peer string) *thread {
	if peer == groupThread {
		return &c.group
	}
	t := c.dms[peer]
	if t == nil {
		t = &thread{}
		c.dms[peer] = t
	}
	return t
}

func (c *ChatStore) route(from, to string) (peer string, self bool) {
	self = from == c.localID
	if to == groupThread {
		return groupThread, self
	}
	if self {
		return to, true
	}
	return from, false
}

func (c *ChatStore) Ingest(from, name, to, text, at string) {
	if text == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	peer, self := c.route(from, to)
	t := c.threadFor(peer)
	if t.add(ChatEntry{From: from, Name: name, Text: text, At: parseAt(at), Self: self, atKey: at}) && !self {
		t.unread++
	}
}

func (c *ChatStore) LoadHistory(msgs []chat.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, m := range msgs {
		if m.Text == "" {
			continue
		}
		peer, self := c.route(m.From, m.To)
		c.threadFor(peer).add(ChatEntry{From: m.From, Text: m.Text, At: parseAt(m.At), Self: self, atKey: m.At})
	}
	c.group.sortAndCap()
	for _, t := range c.dms {
		t.sortAndCap()
	}
}

func (c *ChatStore) GroupEntries() []ChatEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ChatEntry(nil), c.group.log...)
}

func (c *ChatStore) DMEntries(peer string) []ChatEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t := c.dms[peer]; t != nil {
		return append([]ChatEntry(nil), t.log...)
	}
	return nil
}

func (c *ChatStore) GroupUnread() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.group.unread
}

func (c *ChatStore) DMUnread(peer string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t := c.dms[peer]; t != nil {
		return t.unread
	}
	return 0
}

func (c *ChatStore) TotalDMUnread() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, t := range c.dms {
		n += t.unread
	}
	return n
}

func (c *ChatStore) MarkGroupRead() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.group.unread = 0
}

func (c *ChatStore) MarkDMRead(peer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t := c.dms[peer]; t != nil {
		t.unread = 0
	}
}

func parseAt(at string) time.Time {
	if at == "" {
		return time.Now()
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05"} {
		if ts, err := time.Parse(layout, at); err == nil {
			return ts.Local()
		}
	}
	return time.Now()
}
