package tui

import (
	"fmt"
	"strings"

	"statusphere-client/internal/presence"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var wcCond = runewidth.NewCondition()

func init() {
	wcCond.EastAsianWidth = false
}

func wcswidth(s string) int {
	return wcCond.StringWidth(s)
}

var (
	customKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	customValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	customEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	customSepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func padRight(s string, w int) string {
	sw := wcswidth(s)
	if sw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-sw)
}

func BlockCustom(canonicalOrder []string) Block {
	return Block{
		Render: func(d presence.Snapshot) string {
			if len(canonicalOrder) == 0 {
				return ""
			}

			type col struct {
				key string
				val string
				w   int
			}

			cols := make([]col, 0, len(canonicalOrder))
			for _, key := range canonicalOrder {
				val := "—"
				if v, ok := d[key]; ok {
					if s := fmt.Sprintf("%v", v); s != "" {
						val = s
					}
				}
				kw := wcswidth(key)
				vw := wcswidth(val)
				cols = append(cols, col{key: key, val: val, w: max(kw, vw)})
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
