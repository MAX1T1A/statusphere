package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	customKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	customValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	customEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	customSepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func padRight(s string, w int) string {
	sw := runewidth.StringWidth(s)
	if sw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-sw)
}

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
				kw := runewidth.StringWidth(key)
				vw := runewidth.StringWidth(val)
				cols = append(cols, col{key: key, val: val, w: max(kw, vw)})
			}

			if len(cols) == 0 {
				return ""
			}

			sep := customSepStyle.Render(" │ ")

			var keys, vals []string
			for _, c := range cols {
				keys = append(keys, customKeyStyle.Render(padRight(c.key, c.w)))
				if c.val == "—" {
					vals = append(vals, customEmptyStyle.Render(padRight(c.val, c.w)))
				} else {
					vals = append(vals, customValueStyle.Render(padRight(c.val, c.w)))
				}
			}

			return strings.Join(keys, sep) + "\n" + strings.Join(vals, sep)
		},
	}
}
