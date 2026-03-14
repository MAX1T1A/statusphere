package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"statusphere-client/internal/stats"
)

var (
	appName   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	appWindow = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	appLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	sumHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	sumApp    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))
	sumBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	sumTime   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

const (
	sumBarWidth = 8
	sumTopN     = 5
	sumMaxName  = 14
)

func renderSummaryStats(s *stats.Summary) string {
	if s == nil || len(s.Apps) == 0 {
		return ""
	}

	maxSec := 0
	for _, a := range s.Apps {
		if a.Seconds > maxSec {
			maxSec = a.Seconds
		}
	}
	if maxSec == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, sumHeader.Render("screen time · "+s.Period))

	limit := len(s.Apps)
	if limit > sumTopN {
		limit = sumTopN
	}

	nameW := 0
	for _, a := range s.Apps[:limit] {
		if len(a.App) > nameW {
			nameW = len(a.App)
		}
	}
	if nameW > sumMaxName {
		nameW = sumMaxName
	}

	for _, a := range s.Apps[:limit] {
		name := a.App
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}

		filled := (a.Seconds * sumBarWidth) / maxSec
		if filled < 1 {
			filled = 1
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", sumBarWidth-filled)

		h := a.Seconds / 3600
		m := (a.Seconds % 3600) / 60
		var t string
		if h > 0 {
			t = fmt.Sprintf("%dh%dm", h, m)
		} else {
			t = fmt.Sprintf("%dm", m)
		}

		lines = append(lines, fmt.Sprintf("%-*s %s %s",
			nameW,
			sumApp.Render(name),
			sumBar.Render(bar),
			sumTime.Render(t),
		))
	}

	return strings.Join(lines, "\n")
}

func BlockApp(cache *stats.Cache) Block {
	return Block{
		Key: "app",
		Render: func(d map[string]any) string {
			app, _ := d["active_app"].(string)
			win, _ := d["active_window"].(string)
			deviceID, _ := d["device_id"].(string)

			var parts []string

			if app != "" || win != "" {
				var line string
				if app == "" {
					line = appLabel.Render("app ") + appWindow.Render(win)
				} else if win == "" {
					line = appLabel.Render("app ") + appName.Render(app)
				} else {
					line = appLabel.Render("app ") + appName.Render(app) + appWindow.Render(" · "+win)
				}
				parts = append(parts, line)
			}

			if cache != nil && deviceID != "" {
				if s, ok := cache.Get(deviceID).(*stats.Summary); ok && s != nil {
					if st := renderSummaryStats(s); st != "" {
						parts = append(parts, st)
					}
				}
			}

			return strings.Join(parts, "\n\n")
		},
	}
}
