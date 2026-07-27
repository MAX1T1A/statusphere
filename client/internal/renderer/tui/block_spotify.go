package tui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/image/draw"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

var (
	spotArtist = lipgloss.NewStyle().Bold(true).Foreground(cName)
	spotTrack  = lipgloss.NewStyle().Bold(true).Foreground(cValue)
	spotPaused = lipgloss.NewStyle().Foreground(cDim)
	spotDim    = lipgloss.NewStyle().Foreground(cDim)

	progFill = lipgloss.NewStyle().Foreground(cOnline)
	progMark = lipgloss.NewStyle().Foreground(cAccent)
	progRest = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
)

const (
	coverCols = 20
	coverRows = 10
)

type coverEntry struct {
	art  string
	done bool
	ok   bool
}

var coverCache struct {
	sync.Mutex
	m     map[string]*coverEntry
	order []string
}

const coverCacheMax = 48

func fetchCover(url string) string {
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return ""
	}

	img, _, err := image.Decode(buf)
	if err != nil {
		return ""
	}

	h := coverRows * 2
	mid := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.CatmullRom.Scale(mid, mid.Bounds(), img, img.Bounds(), draw.Over, nil)

	dst := image.NewRGBA(image.Rect(0, 0, coverCols, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), mid, mid.Bounds(), draw.Over, nil)

	var sb strings.Builder
	for y := 0; y < h; y += 2 {
		for x := range coverCols {
			tr, tg, tb, _ := dst.At(x, y).RGBA()
			br, bg, bb, _ := dst.At(x, y+1).RGBA()
			fmt.Fprintf(&sb, "\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm▀\033[0m",
				tr>>8, tg>>8, tb>>8,
				br>>8, bg>>8, bb>>8)
		}
		if y < h-2 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

var (
	placeholderOnce sync.Once
	placeholderArt  string
)

func placeholderShade(i, n int) (int, int, int) {
	f := float64(i) / float64(n-1)
	return int(24 + f*20), int(27 + f*22), int(37 + f*30)
}

func placeholderCover() string {
	placeholderOnce.Do(func() {
		var sb strings.Builder
		n := coverRows * 2
		for row := range coverRows {
			tr, tg, tb := placeholderShade(row*2, n)
			br, bg, bb := placeholderShade(row*2+1, n)
			for range coverCols {
				fmt.Fprintf(&sb, "\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm▀\033[0m", tr, tg, tb, br, bg, bb)
			}
			if row < coverRows-1 {
				sb.WriteByte('\n')
			}
		}
		placeholderArt = sb.String()
	})
	return placeholderArt
}

func getCover(url string) string {
	if url == "" {
		return placeholderCover()
	}

	coverCache.Lock()
	if coverCache.m == nil {
		coverCache.m = make(map[string]*coverEntry)
	}
	if e, ok := coverCache.m[url]; ok {
		done, art, good := e.done, e.art, e.ok
		coverCache.Unlock()
		if done && good {
			return art
		}
		return placeholderCover()
	}

	coverCache.m[url] = &coverEntry{}
	coverCache.order = append(coverCache.order, url)
	if len(coverCache.order) > coverCacheMax {
		oldest := coverCache.order[0]
		coverCache.order = coverCache.order[1:]
		delete(coverCache.m, oldest)
	}
	coverCache.Unlock()

	go func() {
		art := fetchCover(url)
		coverCache.Lock()
		if e, ok := coverCache.m[url]; ok {
			e.art = art
			e.done = true
			e.ok = art != ""
		}
		coverCache.Unlock()
	}()

	return placeholderCover()
}

func fmtClock(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func trackLength(d presence.Snapshot) int {
	length64, _ := d.Int(presence.KeySpotifyLength)
	return int(length64)
}

func currentPosition(d presence.Snapshot) (int, int) {
	pos64, _ := d.Int(presence.KeySpotifyPosition)
	pos := int(pos64)
	length := trackLength(d)
	if d.String(presence.KeySpotifyStatus) == "playing" {
		if seen, ok := d.Int(presence.KeyLastSeen); ok {
			pos += int(time.Now().Unix() - seen)
		}
	}
	if pos < 0 {
		pos = 0
	}
	if length > 0 && pos > length {
		pos = length
	}
	return pos, length
}

func renderProgress(d presence.Snapshot) string {
	pos, length := currentPosition(d)
	if length <= 0 {
		return ""
	}
	const width = 14
	filled := max(0, min(pos*width/length, width))
	bar := progFill.Render(strings.Repeat("━", filled)) +
		progMark.Render("●") +
		progRest.Render(strings.Repeat("─", width-filled))
	return spotDim.Render(fmtClock(pos)) + " " + bar + " " + spotDim.Render(fmtClock(length))
}

var spotBarColors = []string{"#4ade80", "#34d399", "#2dd4bf", "#22d3ee", "#38bdf8", "#60a5fa", "#818cf8"}

func renderSpotifyStats(s *stats.SpotifyStats) string {
	if s == nil || s.TotalSeconds == 0 {
		return ""
	}

	var lines []string

	h := s.TotalSeconds / 3600
	m := (s.TotalSeconds % 3600) / 60
	if h > 0 {
		lines = append(lines, fmt.Sprintf("listened %dh %dm", h, m))
	} else {
		lines = append(lines, fmt.Sprintf("listened %dm", m))
	}

	if len(s.Daily) > 1 {
		lines = append(lines, "")

		maxSec := 0
		for _, d := range s.Daily {
			if d.Seconds > maxSec {
				maxSec = d.Seconds
			}
		}
		if maxSec == 0 {
			maxSec = 1
		}

		barWidth := 8
		weekdays := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
		dimBar := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

		for idx, d := range s.Daily {
			label := d.Day
			if len(d.Day) >= 10 {
				t, err := time.Parse("2006-01-02", d.Day)
				if err == nil {
					dow := int(t.Weekday())
					if dow == 0 {
						dow = 6
					} else {
						dow--
					}
					label = weekdays[dow]
				}
			}

			filled := (d.Seconds * barWidth) / maxSec
			if filled < 1 && d.Seconds > 0 {
				filled = 1
			}

			color := spotBarColors[idx%len(spotBarColors)]
			barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

			bar := barStyle.Render(strings.Repeat("█", filled)) + dimBar.Render(strings.Repeat("░", barWidth-filled))
			lines = append(lines, labelStyle.Render(label)+" "+bar+" "+spotDim.Render(fmt.Sprintf("%dm", d.Seconds/60)))
		}
	}

	return strings.Join(lines, "\n")
}

func topDur(sec int) string {
	m := sec / 60
	if m >= 60 {
		return fmt.Sprintf("%dh%02dm", m/60, m%60)
	}
	return fmt.Sprintf("%dm", m)
}

func padLine(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	if gap := w - lipgloss.Width(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

const topN = 3

func topTracksLines(s *stats.SpotifyStats) []string {
	if s == nil || len(s.TopTracks) == 0 {
		return nil
	}
	idxStyle := lipgloss.NewStyle().Foreground(cDim)
	lines := []string{spotDim.Render("top tracks")}
	for i, t := range s.TopTracks {
		if i >= topN {
			break
		}
		lines = append(lines, idxStyle.Render(fmt.Sprintf("%d ", i+1))+spotTrack.Render(t.Title)+spotDim.Render("  "+topDur(t.Seconds)))
	}
	return lines
}

func topArtistsLines(s *stats.SpotifyStats) []string {
	if s == nil || len(s.TopArtists) == 0 {
		return nil
	}
	idxStyle := lipgloss.NewStyle().Foreground(cDim)
	lines := []string{spotDim.Render("top artists")}
	for i, a := range s.TopArtists {
		if i >= topN {
			break
		}
		lines = append(lines, idxStyle.Render(fmt.Sprintf("%d ", i+1))+spotArtist.Render(a.Artist)+spotDim.Render("  "+topDur(a.Seconds)))
	}
	return lines
}

func coverBesideStats(art, statsText string) []string {
	if art == "" && statsText == "" {
		return nil
	}
	if statsText == "" {
		return strings.Split(art, "\n")
	}
	artLines := strings.Split(art, "\n")
	statLines := strings.Split(statsText, "\n")
	rows := max(len(artLines), len(statLines))
	pad := strings.Repeat(" ", coverCols)
	var out []string
	for i := range rows {
		l := pad
		if i < len(artLines) {
			l = artLines[i]
		}
		r := ""
		if i < len(statLines) {
			r = statLines[i]
		}
		out = append(out, l+"  "+r)
	}
	return out
}

func zipColumns(left, right []string, gap int) string {
	leftW := 0
	for _, l := range left {
		if w := lipgloss.Width(l); w > leftW {
			leftW = w
		}
	}
	sep := strings.Repeat(" ", gap)
	rows := max(len(left), len(right))
	var out []string
	for i := range rows {
		var l, r string
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if r == "" {
			out = append(out, l)
		} else {
			out = append(out, padLine(l, leftW)+sep+r)
		}
	}
	return strings.Join(out, "\n")
}

func BlockSpotify(cache *stats.Cache) Block {
	return Block{
		Render: func(d presence.Snapshot) string {
			display := d.String(presence.KeySpotifyDisplay)
			if display == "" {
				display = d.String("music")
			}
			if display == "" {
				return sectionLabel("music") + spotDim.Render("—")
			}

			line := sectionLabel("music") + spotTrack.Render(display)
			if d.String(presence.KeySpotifyStatus) == "paused" {
				line += spotPaused.Render(" (paused)")
			}
			if cache != nil && d.DeviceID() != "" {
				if s, ok := cache.Get(d.DeviceID()).(*stats.SpotifyStats); ok && s != nil && s.TotalSeconds > 0 {
					line += spotDim.Render("  ·  " + durShort(s.TotalSeconds) + " this week")
				}
			}
			return line
		},
	}
}

func linesWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		if lw := lipgloss.Width(l); lw > w {
			w = lw
		}
	}
	return w
}

func spotifyDetail(d presence.Snapshot, cache *stats.Cache, coListeners []string, width int) string {
	display := d.String(presence.KeySpotifyDisplay)
	if display == "" {
		return ""
	}

	status := d.String(presence.KeySpotifyStatus)
	artist := d.String(presence.KeySpotifyArtist)
	track := d.String(presence.KeySpotifyTrack)
	album := d.String(presence.KeySpotifyAlbum)
	artURL := d.String(presence.KeySpotifyArtURL)

	var icon string
	switch status {
	case "playing":
		icon = "▶ "
	case "paused":
		icon = "⏸ "
	default:
		icon = "♪ "
	}

	var head []string
	if status == "paused" {
		head = append(head, spotPaused.Render(icon+display))
	} else if artist != "" && track != "" {
		head = append(head, icon+spotArtist.Render(artist))
		head = append(head, "  "+spotTrack.Render(track))
	} else {
		head = append(head, icon+spotTrack.Render(display))
	}
	if album != "" && album != track {
		head = append(head, "  "+spotDim.Render(album))
	}
	if prog := renderProgress(d); prog != "" {
		head = append(head, "  "+prog)
	}
	if len(coListeners) > 0 {
		head = append(head, "  "+spotDim.Render("· with "+strings.Join(coListeners, ", ")+" now"))
	}

	var s *stats.SpotifyStats
	if cache != nil && d.DeviceID() != "" {
		if got, ok := cache.Get(d.DeviceID()).(*stats.SpotifyStats); ok {
			s = got
		}
	}

	art := getCover(artURL)
	statsText := renderSpotifyStats(s)

	left := append([]string{}, head...)
	if cover := coverBesideStats(art, statsText); len(cover) > 0 {
		left = append(left, "")
		left = append(left, cover...)
	}

	right := topTracksLines(s)
	if artists := topArtistsLines(s); len(artists) > 0 {
		if len(right) > 0 {
			right = append(right, "")
		}
		right = append(right, artists...)
	}

	leftW := linesWidth(left)
	if len(right) > 0 && leftW+4+linesWidth(right) <= width {
		if ly := renderLyricsLines(d, max(width-leftW-4, 12)); len(ly) > 0 {
			right = append(right, "")
			right = append(right, ly...)
		}
		return zipColumns(left, right, 4)
	}

	stacked := append([]string{}, left...)
	if len(right) > 0 {
		stacked = append(stacked, "")
		stacked = append(stacked, right...)
	}
	if ly := renderLyricsLines(d, width); len(ly) > 0 {
		stacked = append(stacked, "")
		stacked = append(stacked, ly...)
	}
	return strings.Join(stacked, "\n")
}
