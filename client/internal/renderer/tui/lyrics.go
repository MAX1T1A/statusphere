package tui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"statusphere-client/internal/presence"
)

type lrcLine struct {
	at   int
	text string
}

type lyricsEntry struct {
	lines []lrcLine
	done  bool
}

var lyricsCache struct {
	sync.Mutex
	m     map[string]*lyricsEntry
	order []string
}

const lyricsCacheMax = 32

var (
	lyricNow  = lipgloss.NewStyle().Bold(true).Foreground(cOnline)
	lyricSoft = lipgloss.NewStyle().Foreground(cDim)
	lrcTimeRe = regexp.MustCompile(`\[(\d+):(\d+)(?:[.:]\d+)?\]`)
)

func lyricsKey(artist, track string) string { return artist + "\x00" + track }

func getLyrics(artist, track, album string, duration int) []lrcLine {
	if artist == "" || track == "" {
		return nil
	}
	key := lyricsKey(artist, track)

	lyricsCache.Lock()
	if lyricsCache.m == nil {
		lyricsCache.m = make(map[string]*lyricsEntry)
	}
	if e, ok := lyricsCache.m[key]; ok {
		lines, done := e.lines, e.done
		lyricsCache.Unlock()
		if done {
			return lines
		}
		return nil
	}
	lyricsCache.m[key] = &lyricsEntry{}
	lyricsCache.order = append(lyricsCache.order, key)
	if len(lyricsCache.order) > lyricsCacheMax {
		oldest := lyricsCache.order[0]
		lyricsCache.order = lyricsCache.order[1:]
		delete(lyricsCache.m, oldest)
	}
	lyricsCache.Unlock()

	go func() {
		lines := fetchLyrics(artist, track, album, duration)
		lyricsCache.Lock()
		if e, ok := lyricsCache.m[key]; ok {
			e.lines = lines
			e.done = true
		}
		lyricsCache.Unlock()
	}()

	return nil
}

func fetchLyrics(artist, track, album string, duration int) []lrcLine {
	client := &http.Client{Timeout: 5 * time.Second}
	if synced := lrclibGet(client, artist, track, album, duration); synced != "" {
		return parseLRC(synced)
	}
	if synced := lrclibSearch(client, artist, track); synced != "" {
		return parseLRC(synced)
	}
	return nil
}

func lrclibBody(client *http.Client, endpoint string, q url.Values) []byte {
	req, err := http.NewRequest("GET", "https://lrclib.net"+endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "statusphere (https://github.com/MAX1T1A/statusphere)")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil
	}
	return buf.Bytes()
}

func lrclibGet(client *http.Client, artist, track, album string, duration int) string {
	q := url.Values{}
	q.Set("artist_name", artist)
	q.Set("track_name", track)
	if album != "" {
		q.Set("album_name", album)
	}
	if duration > 0 {
		q.Set("duration", strconv.Itoa(duration))
	}
	body := lrclibBody(client, "/api/get", q)
	if body == nil {
		return ""
	}
	var d struct {
		SyncedLyrics string `json:"syncedLyrics"`
	}
	if err := json.Unmarshal(body, &d); err != nil {
		return ""
	}
	return d.SyncedLyrics
}

func lrclibSearch(client *http.Client, artist, track string) string {
	q := url.Values{}
	q.Set("track_name", track)
	q.Set("artist_name", artist)
	body := lrclibBody(client, "/api/search", q)
	if body == nil {
		return ""
	}
	var arr []struct {
		SyncedLyrics string `json:"syncedLyrics"`
	}
	if err := json.Unmarshal(body, &arr); err != nil {
		return ""
	}
	for _, r := range arr {
		if r.SyncedLyrics != "" {
			return r.SyncedLyrics
		}
	}
	return ""
}

func parseLRC(s string) []lrcLine {
	var lines []lrcLine
	for _, raw := range strings.Split(s, "\n") {
		stamps := lrcTimeRe.FindAllStringSubmatch(raw, -1)
		if len(stamps) == 0 {
			continue
		}
		text := strings.TrimSpace(lrcTimeRe.ReplaceAllString(raw, ""))
		if text == "" {
			continue
		}
		for _, st := range stamps {
			mm, _ := strconv.Atoi(st[1])
			ss, _ := strconv.Atoi(st[2])
			lines = append(lines, lrcLine{at: mm*60 + ss, text: text})
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].at < lines[j].at })
	return lines
}

func currentLyricIndex(lines []lrcLine, pos int) int {
	idx := -1
	for i, l := range lines {
		if l.at > pos {
			break
		}
		idx = i
	}
	return idx
}

func renderLyricsLines(d presence.Snapshot, width int) []string {
	lines := getLyrics(
		d.String(presence.KeySpotifyArtist),
		d.String(presence.KeySpotifyTrack),
		d.String(presence.KeySpotifyAlbum),
		trackLength(d),
	)
	if len(lines) == 0 {
		return nil
	}

	pos, _ := currentPosition(d)
	idx := currentLyricIndex(lines, pos)

	const window = 5
	lo := max(idx-1, 0)
	if idx < 0 {
		lo = 0
	}
	hi := min(lo+window-1, len(lines)-1)
	lo = max(hi-(window-1), 0)

	textW := max(width, 8)
	var out []string
	for i := lo; i <= hi; i++ {
		t := ansi.Truncate(lines[i].text, textW, "…")
		if i == idx {
			out = append(out, lyricNow.Render(t))
		} else {
			out = append(out, lyricSoft.Render(t))
		}
	}
	return out
}
