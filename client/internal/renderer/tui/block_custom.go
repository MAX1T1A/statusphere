package tui

import (
	"fmt"
	"strings"

	"statusphere-client/internal/presence"

	"github.com/charmbracelet/lipgloss"
)

var (
	customKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	customValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	customSepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func fieldValue(d presence.Snapshot, key string) string {
	if v, ok := d[key]; ok {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}

func BlockCustom() Block {
	return Block{
		Render: func(d presence.Snapshot) string {
			fields := d.Strings(presence.KeyCustomFields)
			if len(fields) == 0 {
				return ""
			}

			var parts []string
			for _, key := range fields {
				val := fieldValue(d, key)
				if val == "" {
					continue
				}
				parts = append(parts, customKeyStyle.Render(key)+" "+customValueStyle.Render(val))
			}
			if len(parts) == 0 {
				return ""
			}
			return sectionLabel("cfg") + strings.Join(parts, customSepStyle.Render("  ·  "))
		},
	}
}
