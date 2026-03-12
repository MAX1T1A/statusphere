package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func ColMemory() Column {
	right := lipgloss.NewStyle().Align(lipgloss.Right).Padding(0, 1)

	return Column{
		Header: "Memory",
		Format: func(d map[string]any) string {
			used, ok1 := d["memory_used_mb"].(float64)
			total, ok2 := d["memory_total_mb"].(float64)
			if !ok1 || !ok2 {
				return "—"
			}
			pct := 0.0
			if total > 0 {
				pct = (used / total) * 100
			}
			return fmt.Sprintf("%.0f/%.0f MB (%.0f%%)", used, total, pct)
		},
		Style: func(string) lipgloss.Style { return right },
	}
}
