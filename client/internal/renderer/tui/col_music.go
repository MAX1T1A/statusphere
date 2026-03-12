package tui

import "github.com/charmbracelet/lipgloss"

func ColMusic() Column {
	return Column{
		Header: "Music",
		Format: func(d map[string]any) string {
			track, _ := d["music"].(string)
			if track == "" {
				return "—"
			}
			return truncate("♪ "+track, 35)
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Padding(0, 1)
		},
	}
}
