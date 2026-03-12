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
		Header: "Status",
		Format: func(d map[string]any) string {
			ts, ok := d["last_seen"].(int64)
			if !ok {
				return "—"
			}
			ago := time.Now().Unix() - ts
			switch {
			case ago < onlineThreshold:
				return "online"
			case ago < idleThreshold:
				return "idle"
			default:
				return "offline"
			}
		},
		Style: func(val string) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1)
			switch val {
			case "online":
				return base.Foreground(lipgloss.Color("10")).Bold(true)
			case "idle":
				return base.Foreground(lipgloss.Color("11"))
			default:
				return base.Foreground(lipgloss.Color("8"))
			}
		},
	}
}
