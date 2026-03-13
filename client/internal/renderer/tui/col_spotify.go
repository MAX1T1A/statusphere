package tui

import "github.com/charmbracelet/lipgloss"

var (
	spotifyArtist = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	spotifyTrack  = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	spotifyPaused = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func ColSpotify() Column {
	return Column{
		Header: "Spotify",
		Format: func(d map[string]any) string {
			display, _ := d["spotify_display"].(string)
			if display == "" {
				return "—"
			}

			status, _ := d["spotify_status"].(string)

			artist, _ := d["spotify_artist"].(string)
			track, _ := d["spotify_track"].(string)

			var icon string
			switch status {
			case "playing":
				icon = "▶ "
			case "paused":
				icon = "⏸ "
				return spotifyPaused.Render(icon + display)
			default:
				icon = "♪ "
			}

			if artist != "" && track != "" {
				return icon + spotifyArtist.Render(artist) + spotifyTrack.Render(" — "+track)
			}
			return icon + spotifyTrack.Render(display)
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		},
	}
}
