package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

func ColPackages() Column {
	return Column{
		Header: "Packages",
		Format: func(d map[string]any) string {
			if v, ok := d["package_count"].(float64); ok {
				return fmt.Sprintf("%.0f", v)
			}
			if v, ok := d["package_count"].(int); ok {
				return strconv.Itoa(v)
			}
			return "—"
		},
		Style: func(string) lipgloss.Style {
			return lipgloss.NewStyle().Align(lipgloss.Right).Padding(0, 1)
		},
	}
}
