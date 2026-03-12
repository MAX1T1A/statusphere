package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func ColLoad() Column {
	right := lipgloss.NewStyle().Align(lipgloss.Right).Padding(0, 1)

	return Column{
		Header: "Load 1m",
		Format: func(d map[string]any) string {
			if v, ok := d["load_avg_1m"].(float64); ok {
				return fmt.Sprintf("%.2f", v)
			}
			return "—"
		},
		Style: func(string) lipgloss.Style { return right },
	}
}
