package tui

import "github.com/charmbracelet/lipgloss"

func ColApp() Column {
	return Column{
		Header: "App",
		Format: func(d map[string]any) string {
			app, _ := d["active_app"].(string)
			win, _ := d["active_window"].(string)
			if app != "" && win != "" {
				return truncate(app+" — "+win, 30)
			}
			if app != "" {
				return truncate(app, 30)
			}
			if win != "" {
				return truncate(win, 30)
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(0, 1)
		},
	}
}
