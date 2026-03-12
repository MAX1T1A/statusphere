package tui

import "github.com/charmbracelet/lipgloss"

func ColApp() Column {
	return Column{
		Header: "App",
		Format: func(d map[string]any) string {
			app, _ := d["active_app"].(string)
			win, _ := d["active_window"].(string)
			if app != "" && win != "" {
				return app + " — " + win
			}
			if app != "" {
				return app
			}
			if win != "" {
				return win
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1)
		},
	}
}
