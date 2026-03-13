package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	appName   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	appWindow = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	appLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func BlockApp() Block {
	return Block{
		Key: "app",
		Render: func(d map[string]any) string {
			app, _ := d["active_app"].(string)
			win, _ := d["active_window"].(string)
			if app == "" && win == "" {
				return ""
			}
			if app == "" {
				return appLabel.Render("app ") + appWindow.Render(win)
			}
			if win == "" {
				return appLabel.Render("app ") + appName.Render(app)
			}
			return appLabel.Render("app ") + appName.Render(app) + appWindow.Render(" · "+win)
		},
	}
}
