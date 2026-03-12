package tui

import "github.com/charmbracelet/lipgloss"

func ColStatus() Column {
	return Column{
		Header: "Status",
		Format: func(d map[string]any) string {
			if v, ok := d["status"].(string); ok {
				return v
			}
			return "—"
		},
		Style: func(val string) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)
			switch val {
			case "online":
				return base.Foreground(lipgloss.Color("10")).Bold(true)
			case "idle":
				return base.Foreground(lipgloss.Color("11"))
			default:
				return base.Foreground(lipgloss.Color("8"))
			}
		},
	}
}
