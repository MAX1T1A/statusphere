package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

var (
	appName   = lipgloss.NewStyle().Bold(true).Foreground(cName)
	appWindow = lipgloss.NewStyle().Foreground(cDim)
	appLabel  = lipgloss.NewStyle().Foreground(cDim)

	sumHeader = lipgloss.NewStyle().Foreground(cLabel)
	sumTime   = lipgloss.NewStyle().Foreground(cDim)
)

var barColors = []string{"#c084fc", "#a78bfa", "#818cf8", "#7dd3fc", "#67e8f9", "#5eead4", "#a5b4fc"}

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

	for i, a := range s.Apps[:limit] {
		name := a.App
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		padded := name + strings.Repeat(" ", nameW-len([]rune(name)))

		filled := (a.Seconds * sumBarWidth) / maxSec
		if filled < 1 {
			filled = 1
		}

		color := barColors[i%len(barColors)]
		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		dimBar := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))

		bar := barStyle.Render(strings.Repeat("█", filled)) + dimBar.Render(strings.Repeat("░", sumBarWidth-filled))

		h := a.Seconds / 3600
		m := (a.Seconds % 3600) / 60
		var t string
		if h > 0 {
			t = fmt.Sprintf("%dh%dm", h, m)
		} else {
			t = fmt.Sprintf("%dm", m)
		}

		lines = append(lines, nameStyle.Render(padded)+" "+bar+" "+sumTime.Render(t))
	}

	return strings.Join(lines, "\n")
}

const maxWindowTitle = 200

func BlockApp(cache *stats.Cache) Block {
	return Block{
		Render: func(d presence.Snapshot) string {
			app := d.String(presence.KeyActiveApp)
			win := d.String(presence.KeyActiveWindow)
			workspace := d.String(presence.KeyActiveWorkspace)

			if app == "" && win == "" {
				return sectionLabel("app") + appWindow.Render("—")
			}

			var val string
			switch {
			case app == "":
				val = appWindow.Render(win)
			case win == "":
				val = appName.Render(app)
			default:
				val = appName.Render(app) + appWindow.Render(" · "+win)
			}
			if workspace != "" {
				val += appLabel.Render(" · ws " + workspace)
			}

			line := sectionLabel("app") + val
			if cache != nil && d.DeviceID() != "" {
				if s, ok := cache.Get(d.DeviceID()).(*stats.Summary); ok && s != nil {
					total := 0
					for _, a := range s.Apps {
						total += a.Seconds
					}
					if total > 0 {
						line += sumTime.Render("  ·  " + durShort(total) + " today")
					}
				}
			}
			return line
		},
	}
}

func appDetail(d presence.Snapshot, cache *stats.Cache) string {
	var parts []string

	app := d.String(presence.KeyActiveApp)
	win := d.String(presence.KeyActiveWindow)
	if len(win) > maxWindowTitle {
		runes := []rune(win)
		win = string(runes[:maxWindowTitle-1]) + "…"
	}
	if app != "" || win != "" {
		line := appName.Render(app)
		if win != "" {
			if app != "" {
				line += appWindow.Render(" · ")
			}
			line += appWindow.Render(win)
		}
		parts = append(parts, line)
	}

	if cache != nil && d.DeviceID() != "" {
		if s, ok := cache.Get(d.DeviceID()).(*stats.Summary); ok && s != nil {
			if st := renderSummaryStats(s); st != "" {
				parts = append(parts, st)
			}
		}
	}
	return strings.Join(parts, "\n\n")
}
