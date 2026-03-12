package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	appNameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	appDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func ColApp() Column {
	return Column{
		Header: "App",
		Format: func(d map[string]any) string {
			app, _ := d["active_app"].(string)
			win, _ := d["active_window"].(string)

			if app == "" && win == "" {
				return "—"
			}
			if app == "" {
				return appDimStyle.Render(win)
			}
			if win == "" {
				return appNameStyle.Render(app)
			}

			return appNameStyle.Render(app) + appDimStyle.Render(" · "+win)
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		},
	}
}
