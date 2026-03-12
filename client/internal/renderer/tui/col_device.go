package tui

import "github.com/charmbracelet/lipgloss"

func ColDevice() Column {
	return Column{
		Header: "Device",
		Format: func(d map[string]any) string {
			if name, ok := d["device_name"].(string); ok && name != "" {
				return name
			}
			if v, ok := d["device_id"].(string); ok {
				return v
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Padding(0, 1)
		},
	}
}
