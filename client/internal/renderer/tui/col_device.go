package tui

import "github.com/charmbracelet/lipgloss"

const maxDeviceLen = 16

func ColDevice() Column {
	return Column{
		Header: "Device",
		Format: func(d map[string]any) string {
			if name, ok := d["device_name"].(string); ok && name != "" {
				return truncate(name, maxDeviceLen)
			}
			if v, ok := d["device_id"].(string); ok {
				return truncate(v, maxDeviceLen)
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Padding(0, 1)
		},
	}
}
