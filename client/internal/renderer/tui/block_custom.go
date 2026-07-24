package tui

import (
	"fmt"
	"strings"

	"statusphere-client/internal/presence"

	"github.com/charmbracelet/lipgloss"
)

var (
	customKeyStyle   = lipgloss.NewStyle().Foreground(cLabel)
	customValueStyle = lipgloss.NewStyle().Bold(true).Foreground(cValue)
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

			var lines []string
			for _, key := range fields {
				val := fieldValue(d, key)
				if val == "" {
					continue
				}
				label := customKeyStyle.Render(fmt.Sprintf("%-*s ", labelWidth, key))
				lines = append(lines, label+customValueStyle.Render(val))
			}
			return strings.Join(lines, "\n")
		},
	}
}
