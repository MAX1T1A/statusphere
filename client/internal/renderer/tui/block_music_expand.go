package tui

import (
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

	for _, block := range extras {
		if len(body) > 0 {
			body = append(body, "")
		}
		body = append(body, block...)
	}

	if len(body) == 0 {
		return nil
	}

	for i, l := range body {
		body[i] = ansi.Truncate(l, inner, "…")
	}
	panel := musicPanelBorder.Width(inner).Render(strings.Join(body, "\n"))
	return indentLines(strings.Split(panel, "\n"), 2)
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
