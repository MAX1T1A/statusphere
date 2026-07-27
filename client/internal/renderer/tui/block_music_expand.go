package tui

import (
	"strings"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

type musicExpandCtx struct {
	cache       *stats.Cache
	coListeners []string
	width       int
}

func musicExpansion(d presence.Snapshot, account string, e *expandState, ctx musicExpandCtx) []string {
	if e == nil || account == "" || d.String(presence.KeySpotifyDisplay) == "" {
		return nil
	}

	var s *stats.SpotifyStats
	needStats := e.enabled(account, "tops") || e.enabled(account, "weekly")
	if needStats && ctx.cache != nil && d.DeviceID() != "" {
		if got, ok := ctx.cache.Get(d.DeviceID()).(*stats.SpotifyStats); ok {
			s = got
		}
	}

	var out []string
	add := func(lines ...string) {
		for _, l := range lines {
			if l != "" {
				out = append(out, l)
			}
		}
	}

	if e.enabled(account, "progress") {
		add(renderProgress(d))
	}
	if e.enabled(account, "album") {
		if album := d.String(presence.KeySpotifyAlbum); album != "" && album != d.String(presence.KeySpotifyTrack) {
			add(spotDim.Render(album))
		}
	}
	if e.enabled(account, "together") && len(ctx.coListeners) > 0 {
		add(spotDim.Render("· with " + strings.Join(ctx.coListeners, ", ") + " now"))
	}
	if e.enabled(account, "cover") {
		if art := getCover(d.String(presence.KeySpotifyArtURL)); art != "" {
			out = append(out, strings.Split(strings.TrimRight(art, "\n"), "\n")...)
		}
	}
	if e.enabled(account, "weekly") {
		if st := renderSpotifyStats(s); st != "" {
			out = append(out, strings.Split(st, "\n")...)
		}
	}
	if e.enabled(account, "tops") {
		out = append(out, topTracksLines(s)...)
		out = append(out, topArtistsLines(s)...)
	}
	if e.enabled(account, "lyrics") {
		out = append(out, renderLyricsLines(d, max(ctx.width, 12))...)
	}

	if len(out) == 0 {
		return nil
	}
	return indentLines(out, 2)
}
