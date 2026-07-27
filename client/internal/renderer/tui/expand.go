package tui

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"statusphere-client/internal/config"
)

const expandFile = "card_blocks.json"

type piece struct {
	id    string
	label string
	desc  string
}

var musicPieces = []piece{
	{"cover", "Album art", "cover of what's playing"},
	{"progress", "Progress", "position in the track"},
	{"album", "Album", "album name"},
	{"together", "Listening together", "who's on the same track"},
	{"lyrics", "Lyrics", "synced, follows the track"},
	{"tracks", "Top tracks", "this week"},
	{"artists", "Top artists", "this week"},
	{"weekly", "Weekly chart", "daily listening"},
}

type expandState struct {
	mu sync.Mutex
	on map[string]map[string]bool
}

func newExpandState() *expandState {
	e := &expandState{on: map[string]map[string]bool{}}
	e.load()
	return e
}

func (e *expandState) enabled(account, id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.on[account][id]
}

func (e *expandState) any(account string, ps []piece) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, p := range ps {
		if e.on[account][p.id] {
			return true
		}
	}
	return false
}

func (e *expandState) toggle(account, id string) {
	e.mu.Lock()
	if e.on[account] == nil {
		e.on[account] = map[string]bool{}
	}
	if e.on[account][id] {
		delete(e.on[account], id)
		if len(e.on[account]) == 0 {
			delete(e.on, account)
		}
	} else {
		e.on[account][id] = true
	}
	e.mu.Unlock()
	e.save()
}

func (e *expandState) load() {
	data, err := config.Read(expandFile)
	if err != nil {
		return
	}
	var stored map[string][]string
	if json.Unmarshal(data, &stored) != nil {
		return
	}
	for account, ids := range stored {
		for _, id := range ids {
			if e.on[account] == nil {
				e.on[account] = map[string]bool{}
			}
			e.on[account][id] = true
		}
	}
}

func (e *expandState) save() {
	e.mu.Lock()
	stored := make(map[string][]string, len(e.on))
	for account, ids := range e.on {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		sort.Strings(list)
		stored[account] = list
	}
	e.mu.Unlock()

	data, err := json.Marshal(stored)
	if err != nil {
		return
	}
	_ = config.Write(expandFile, data, 0o600)
}

func checkbox(on bool) string {
	if on {
		return accentStyle.Render("[x] ")
	}
	return dimStyle.Render("[ ] ")
}

func indentLines(lines []string, by int) []string {
	pad := strings.Repeat(" ", by)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			out = append(out, "")
			continue
		}
		out = append(out, pad+l)
	}
	return out
}
