package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	onlineThreshold = 10
	idleThreshold   = 60
)

func ColStatus() Column {
	return Column{
		Header: "●",
		Format: func(d map[string]any) string {
			ts, ok := d["last_seen"].(int64)
			if !ok {
				return "○"
			}
			ago := timeNow() - ts
			switch {
			case ago < onlineThreshold:
				return "●"
			case ago < idleThreshold:
				return "◐"
			default:
				return "○"
			}
		},
		Style: func(val string) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)
			switch val {
			case "●":
				return base.Foreground(lipgloss.Color("10"))
			case "◐":
				return base.Foreground(lipgloss.Color("11"))
			default:
				return base.Foreground(lipgloss.Color("8"))
			}
		},
	}
}

func timeNow() int64 {
	return time.Now().Unix()
}
