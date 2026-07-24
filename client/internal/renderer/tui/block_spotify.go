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
	"golang.org/x/image/draw"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

var (
	spotArtist = lipgloss.NewStyle().Bold(true).Foreground(cName)
	spotTrack  = lipgloss.NewStyle().Bold(true).Foreground(cValue)
	spotPaused = lipgloss.NewStyle().Foreground(cDim)
	spotDim    = lipgloss.NewStyle().Foreground(cDim)
)

var artCache struct {
	sync.Mutex
	url string
	art string
}

const (
	coverCols = 20
	coverRows = 10
)

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

func getCover(url string) string {
	if url == "" {
		return ""
	}

	artCache.Lock()
	if artCache.url == url && artCache.art != "" {
		result := artCache.art
		artCache.Unlock()
		return result
	}
	artCache.Unlock()

	art := fetchCover(url)
	if art == "" {
		return ""
	}

	artCache.Lock()
	artCache.url = url
	artCache.art = art
	artCache.Unlock()

	return art
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

		barWidth := 8
		weekdays := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
		dimBar := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

		idx := 0
		for _, d := range s.Daily {
			if d.Seconds == 0 {
				continue
			}

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
			if filled < 1 {
				filled = 1
			}

			color := spotBarColors[idx%len(spotBarColors)]
			barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

			bar := barStyle.Render(strings.Repeat("█", filled)) + dimBar.Render(strings.Repeat("░", barWidth-filled))
			lines = append(lines, labelStyle.Render(label)+" "+bar+" "+spotDim.Render(fmt.Sprintf("%dm", d.Seconds/60)))
			idx++
		}
	}

	return strings.Join(lines, "\n")
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
					line += spotDim.Render("  ·  " + durShort(s.TotalSeconds) + "/wk")
				}
			}
			return line
		},
	}
}

func spotifyDetail(d presence.Snapshot, cache *stats.Cache) string {
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

	var lines []string
	if status == "paused" {
		lines = append(lines, spotPaused.Render(icon+display))
	} else if artist != "" && track != "" {
		lines = append(lines, icon+spotArtist.Render(artist))
		lines = append(lines, "  "+spotTrack.Render(track))
	} else {
		lines = append(lines, icon+spotTrack.Render(display))
	}
	if album != "" && album != track {
		lines = append(lines, "  "+spotDim.Render(album))
	}

	text := strings.Join(lines, "\n")
	art := getCover(artURL)

	var statsText string
	if cache != nil && d.DeviceID() != "" {
		if s, ok := cache.Get(d.DeviceID()).(*stats.SpotifyStats); ok && s != nil {
			statsText = renderSpotifyStats(s)
		}
	}

	var parts []string
	if art != "" && statsText != "" {
		artLines := strings.Split(art, "\n")
		statLines := strings.Split(statsText, "\n")
		maxLines := len(artLines)
		if len(statLines) > maxLines {
			maxLines = len(statLines)
		}
		pad := strings.Repeat(" ", coverCols)
		var combined []string
		for i := range maxLines {
			left := pad
			if i < len(artLines) {
				left = artLines[i]
			}
			right := ""
			if i < len(statLines) {
				right = statLines[i]
			}
			combined = append(combined, left+"  "+right)
		}
		parts = append(parts, strings.Join(combined, "\n"))
	} else if art != "" {
		parts = append(parts, art)
	} else if statsText != "" {
		parts = append(parts, statsText)
	}
	parts = append(parts, text)

	return strings.Join(parts, "\n\n")
}
