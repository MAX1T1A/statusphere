package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	customKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	customValue = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	customEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func BlockCustom() Block {
	return Block{
		Key: "custom",
		Render: func(d map[string]any) string {
			fields, _ := d["custom_fields"].([]any)

			var lines []string
			for _, f := range fields {
				key, ok := f.(string)
				if !ok || key == "" {
					continue
				}
				val, _ := d[key].(string)
				if val != "" {
					lines = append(lines, customKey.Render(key)+" "+customValue.Render(val))
				} else {
					lines = append(lines, customKey.Render(key)+" "+customEmpty.Render("—"))
				}
			}

			if len(lines) == 0 {
				return ""
			}
			return strings.Join(lines, "\n")
		},
	}
}
