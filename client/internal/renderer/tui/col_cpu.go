package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func ColCPU() Column {
	right := lipgloss.NewStyle().Align(lipgloss.Right).Padding(0, 1)

	return Column{
		Header: "CPU",
		Format: func(d map[string]any) string {
			if v, ok := d["cpu_percent"].(float64); ok {
				return fmt.Sprintf("%.1f%%", v)
			}
			return "—"
		},
		Style: func(val string) lipgloss.Style {
			if val == "—" {
				return right
			}
			var v float64
			fmt.Sscanf(val, "%f", &v)
			switch {
			case v < 50:
				return right.Foreground(lipgloss.Color("10"))
			case v < 80:
				return right.Foreground(lipgloss.Color("11"))
			default:
				return right.Foreground(lipgloss.Color("9")).Bold(true)
			}
		},
	}
}
