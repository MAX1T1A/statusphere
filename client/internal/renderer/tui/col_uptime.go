package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func ColUptime() Column {
	return Column{
		Header: "Uptime",
		Format: func(d map[string]any) string {
			if v, ok := d["uptime_hours"].(float64); ok {
				return fmt.Sprintf("%.1fh", v)
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Align(lipgloss.Right).Padding(0, 1)
		},
	}
}
