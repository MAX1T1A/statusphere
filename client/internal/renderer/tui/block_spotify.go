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
)

var (
	spotArtist = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	spotTrack  = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	spotPaused = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	spotDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
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

func BlockSpotify() Block {
	return Block{
		Key: "spotify",
		Render: func(d map[string]any) string {
			display, _ := d["spotify_display"].(string)
			if display == "" {
				music, _ := d["music"].(string)
				if music == "" {
					return ""
				}
				return spotDim.Render("♪ " + music)
			}

			status, _ := d["spotify_status"].(string)
			artist, _ := d["spotify_artist"].(string)
			track, _ := d["spotify_track"].(string)
			album, _ := d["spotify_album"].(string)
			artURL, _ := d["spotify_art_url"].(string)

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
			if art != "" {
				return art + "\n\n" + text
			}
			return text
		},
	}
}
