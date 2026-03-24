package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	customKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	customValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	customEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	customSepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func BlockCustom() Block {
	return Block{
		Key: "custom",
		Render: func(d map[string]any) string {
			fields, _ := d["custom_fields"].([]any)

			type col struct {
				key string
				val string
				w   int
			}

			var cols []col
			for _, f := range fields {
				key, ok := f.(string)
				if !ok || key == "" {
					continue
				}
				val := "—"
				if v, ok := d[key]; ok {
					s := fmt.Sprintf("%v", v)
					if s != "" {
						val = s
					}
				}
				w := len(key)
				if len(val) > w {
					w = len(val)
				}
				cols = append(cols, col{key: key, val: val, w: w})
			}

			if len(cols) == 0 {
				return ""
			}

			sep := customSepStyle.Render(" │ ")

			var keys, vals []string
			for _, c := range cols {
				keys = append(keys, customKeyStyle.Render(c.key+strings.Repeat(" ", c.w-len(c.key))))
				raw := c.val
				pad := strings.Repeat(" ", c.w-len(raw))
				if raw == "—" {
					vals = append(vals, customEmptyStyle.Render(raw+pad))
				} else {
					vals = append(vals, customValueStyle.Render(raw+pad))
				}
			}

			return strings.Join(keys, sep) + "\n" + strings.Join(vals, sep)
		},
	}
}
