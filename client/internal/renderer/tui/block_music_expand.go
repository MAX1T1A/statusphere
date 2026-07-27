package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

var musicPanelBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(cOffline).
	Padding(0, 2)

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
	if (on("tops") || on("weekly")) && ctx.cache != nil && d.DeviceID() != "" {
		if got, ok := ctx.cache.Get(d.DeviceID()).(*stats.SpotifyStats); ok {
			s = got
		}
	}

	inner := max(ctx.width-musicPanelBorder.GetHorizontalFrameSize(), 16)

	var facts []string
	if on("progress") {
		if p := renderProgress(d); p != "" {
			facts = append(facts, p)
		}
	}
	if on("album") {
		if album := d.String(presence.KeySpotifyAlbum); album != "" && album != d.String(presence.KeySpotifyTrack) {
			facts = append(facts, spotDim.Render(album))
		}
	}
	if on("together") && len(ctx.coListeners) > 0 {
		facts = append(facts, spotDim.Render("· with "+strings.Join(ctx.coListeners, ", ")+" now"))
	}

	var cover []string
	if on("cover") {
		if art := getCover(d.String(presence.KeySpotifyArtURL)); art != "" {
			cover = strings.Split(strings.TrimRight(art, "\n"), "\n")
		}
	}

	body := joinSideBySide(cover, facts, inner)

	var extras [][]string
	if on("weekly") {
		if st := renderSpotifyStats(s); st != "" {
			extras = append(extras, strings.Split(st, "\n"))
		}
	}
	if on("tops") {
		tops := append([]string{}, topTracksLines(s)...)
		if artists := topArtistsLines(s); len(artists) > 0 {
			if len(tops) > 0 {
				tops = append(tops, "")
			}
			tops = append(tops, artists...)
		}
		if len(tops) > 0 {
			extras = append(extras, tops)
		}
	}
	if on("lyrics") {
		if ly := renderLyricsLines(d, inner); len(ly) > 0 {
			extras = append(extras, ly)
		}
	}

	blocks := [][]string{}
	if len(body) > 0 {
		blocks = append(blocks, body)
	}
	blocks = append(blocks, extras...)
	body = packColumns(blocks, inner, 4)

	if len(body) == 0 {
		return nil
	}

	for i, l := range body {
		body[i] = ansi.Truncate(l, inner, "…")
	}
	panel := musicPanelBorder.Width(inner).Render(strings.Join(body, "\n"))
	return indentLines(strings.Split(panel, "\n"), 2)
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

	for k := len(kept); k > 1; k-- {
		if cols, ok := balanceColumns(kept, k, width, gap); ok {
			return mergeColumns(cols, gap)
		}
	}
	return mergeColumns([][]string{stack(kept)}, gap)
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
			if len(cols[i]) > 0 {
				cols[i] = append(cols[i], "")
			}
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

func columnsWidth(cols [][]string, gap int) int {
	w := 0
	for i, c := range cols {
		if i > 0 {
			w += gap
		}
		w += linesWidth(c)
	}
	return w
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
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, b...)
	}
	return out
}

func joinSideBySide(left, right []string, width int) []string {
	switch {
	case len(left) == 0:
		return right
	case len(right) == 0:
		return left
	}
	if linesWidth(left)+4+linesWidth(right) > width {
		out := append([]string{}, left...)
		out = append(out, "")
		return append(out, right...)
	}
	return strings.Split(zipColumns(left, right, 4), "\n")
}
