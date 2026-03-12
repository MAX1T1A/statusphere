package tui

import "github.com/charmbracelet/lipgloss"

func ColWorkspace() Column {
	return Column{
		Header: "Workspace",
		Format: func(d map[string]any) string {
			if v, ok := d["active_workspace"].(string); ok {
				return v
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Align(lipgloss.Right).Padding(0, 1)
		},
	}
}
