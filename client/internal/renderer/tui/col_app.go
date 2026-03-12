package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const maxAppLen = 40

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

			if app != "" {
				app = CleanAppName(app)
			}

			if app == "" && win == "" {
				return "—"
			}

			if app == "" {
				return appDimStyle.Render(truncate(win, maxAppLen))
			}

			title := cleanTitle(win, app)

			if title == "" {
				return appNameStyle.Render(truncate(app, maxAppLen))
			}

			appPart := appNameStyle.Render(app)
			titlePart := appDimStyle.Render(" · " + truncate(title, maxAppLen-len([]rune(app))-3))
			return appPart + titlePart
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		},
	}
}

func cleanTitle(title, app string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}

	lower := strings.ToLower(title)
	lowerApp := strings.ToLower(app)

	if strings.HasSuffix(lower, " - "+lowerApp) {
		title = strings.TrimSpace(title[:len(title)-len(app)-3])
	} else if strings.HasSuffix(lower, " — "+lowerApp) {
		title = strings.TrimSpace(title[:len(title)-len(app)-3])
	} else if strings.HasPrefix(lower, lowerApp+" - ") {
		title = strings.TrimSpace(title[len(app)+3:])
	} else if strings.HasPrefix(lower, lowerApp+" — ") {
		title = strings.TrimSpace(title[len(app)+3:])
	}

	title = strings.TrimLeft(title, "- —·")
	title = strings.TrimSpace(title)
	return title
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}
