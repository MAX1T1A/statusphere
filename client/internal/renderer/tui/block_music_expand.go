package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

var musicTileBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(cOffline).
	Padding(0, 1)

const tileGap = 2

type musicExpandCtx struct {
	cache       *stats.Cache
	coListeners []string
	width       int
}

func musicExpansion(d presence.Snapshot, account string, e *expandState, ctx musicExpandCtx) []string {
	if e == nil || account == "" || d.String(presence.KeySpotifyDisplay) == "" {
		return nil
	}
	on := func(id string) bool { return e.enabled(account, id) }

	var s *stats.SpotifyStats
	if (on("tracks") || on("artists") || on("weekly")) && ctx.cache != nil && d.DeviceID() != "" {
		if got, ok := ctx.cache.Get(d.DeviceID()).(*stats.SpotifyStats); ok {
			s = got
		}
	}

	frame := musicTileBorder.GetHorizontalFrameSize()
	body := max(ctx.width-frame, 12)

	var tiles [][]string
	add := func(lines []string) {
		if len(lines) == 0 {
			return
		}
		for i, l := range lines {
			lines[i] = ansi.Truncate(l, body, "…")
		}
		w := linesWidth(lines) + musicTileBorder.GetHorizontalPadding()
		tiles = append(tiles, strings.Split(musicTileBorder.Width(w).Render(strings.Join(lines, "\n")), "\n"))
	}

	if on("cover") {
		if art := getCover(d.String(presence.KeySpotifyArtURL)); art != "" {
			add(strings.Split(strings.TrimRight(art, "\n"), "\n"))
		}
	}
	if on("progress") {
		if p := renderProgress(d); p != "" {
			add([]string{p})
		}
	}
	if on("album") {
		if album := d.String(presence.KeySpotifyAlbum); album != "" && album != d.String(presence.KeySpotifyTrack) {
			add([]string{spotDim.Render(album)})
		}
	}
	if on("together") && len(ctx.coListeners) > 0 {
		add([]string{spotDim.Render("with " + strings.Join(ctx.coListeners, ", ") + " now")})
	}
	if on("weekly") {
		if st := renderSpotifyStats(s); st != "" {
			add(strings.Split(st, "\n"))
		}
	}
	if on("tracks") {
		add(topTracksLines(s))
	}
	if on("artists") {
		add(topArtistsLines(s))
	}
	if on("lyrics") {
		add(renderLyricsLines(d, max(body, 12)))
	}

	if len(tiles) == 0 {
		return nil
	}
	return indentLines(packColumns(tiles, ctx.width, tileGap), 2)
}

func packColumns(blocks [][]string, width, gap int) []string {
	var kept [][]string
	for _, b := range blocks {
		if len(b) > 0 {
			kept = append(kept, b)
		}
	}
	if len(kept) == 0 {
		return nil
	}

	best := [][]string{stack(kept)}
	bestH, bestWaste := len(best[0]), 0

	for k := 2; k <= len(kept); k++ {
		cols, ok := balanceColumns(kept, k, width, gap)
		if !ok {
			continue
		}
		h := 0
		for _, c := range cols {
			h = max(h, len(c))
		}
		waste := 0
		for _, c := range cols {
			waste += h - len(c)
		}
		if h < bestH || (h == bestH && waste < bestWaste) {
			best, bestH, bestWaste = cols, h, waste
		}
	}
	return mergeColumns(best, gap)
}

func balanceColumns(blocks [][]string, k, width, gap int) ([][]string, bool) {
	cols := make([][]string, k)
	widths := make([]int, k)

	prospective := func(target, bw int) int {
		sum, used := 0, 0
		for i := range cols {
			w, occupied := widths[i], len(cols[i]) > 0
			if i == target {
				w, occupied = max(w, bw), true
			}
			if !occupied {
				continue
			}
			if used > 0 {
				sum += gap
			}
			sum += w
			used++
		}
		return sum
	}

	for _, b := range blocks {
		bw := linesWidth(b)
		order := make([]int, k)
		for i := range order {
			order[i] = i
		}
		sort.SliceStable(order, func(a, c int) bool { return len(cols[order[a]]) < len(cols[order[c]]) })

		placed := false
		for _, i := range order {
			if prospective(i, bw) > width {
				continue
			}
			widths[i] = max(widths[i], bw)
			cols[i] = append(cols[i], b...)
			placed = true
			break
		}
		if !placed {
			return nil, false
		}
	}

	var used [][]string
	for _, c := range cols {
		if len(c) > 0 {
			used = append(used, c)
		}
	}
	return used, len(used) == k
}

func mergeColumns(cols [][]string, gap int) []string {
	if len(cols) == 0 {
		return nil
	}
	merged := cols[0]
	for _, next := range cols[1:] {
		merged = strings.Split(zipColumns(merged, next, gap), "\n")
	}
	return merged
}

func stack(blocks [][]string) []string {
	var out []string
	for _, b := range blocks {
		out = append(out, b...)
	}
	return out
}
