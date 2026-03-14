package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	onlineDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	idleDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	offlineDot = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	deviceName = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	uptimeDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func statusDot(d map[string]any) string {
	ts, ok := d["last_seen"].(int64)
	if !ok {
		return offlineDot.Render("○")
	}
	ago := time.Now().Unix() - ts
	switch {
	case ago < 10:
		return onlineDot.Render("●")
	case ago < 60:
		return idleDot.Render("◐")
	default:
		return offlineDot.Render("○")
	}
}

func formatUptime(d map[string]any) string {
	v, ok := d["uptime_hours"].(float64)
	if !ok {
		return ""
	}
	if v < 1 {
		return fmt.Sprintf("%.0fm", v*60)
	}
	if v < 24 {
		return fmt.Sprintf("%.1fh", v)
	}
	return fmt.Sprintf("%.0fd", v/24)
}

func BlockHeader() Block {
	return Block{
		Key: "header",
		Render: func(d map[string]any) string {
			name := "unknown"
			if n, ok := d["device_name"].(string); ok && n != "" {
				name = n
			} else if id, ok := d["device_id"].(string); ok {
				name = id
			}

			line := statusDot(d) + " " + deviceName.Render(name)
			if up := formatUptime(d); up != "" {
				line += uptimeDim.Render(" · " + up)
			}
			return line
		},
	}
}
