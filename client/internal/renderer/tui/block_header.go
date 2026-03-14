package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	onlineDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	idleDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	offlineDot = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	hdrName    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	hdrDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	hdrBatLow  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hdrBatMid  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	hdrBatHi   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func statusDot(d map[string]any) string {
	ts, ok := d["last_seen"].(int64)
	if !ok {
		return offlineDot.Render("○")
	}
	ago := time.Now().Unix() - ts
	switch {
	case ago < 10:
		return onlineDot.Render("●")
	case ago < 60:
		return idleDot.Render("◐")
	default:
		return offlineDot.Render("○")
	}
}

func formatUptime(d map[string]any) string {
	v, ok := d["uptime_hours"].(float64)
	if !ok {
		return ""
	}
	if v < 1 {
		return fmt.Sprintf("%.0fm", v*60)
	}
	if v < 24 {
		return fmt.Sprintf("%.1fh", v)
	}
	return fmt.Sprintf("%.0fd", v/24)
}

func formatBattery(d map[string]any) string {
	pct, ok := d["battery_pct"].(int)
	if !ok {
		return ""
	}

	var style lipgloss.Style
	switch {
	case pct <= 20:
		style = hdrBatLow
	case pct <= 50:
		style = hdrBatMid
	default:
		style = hdrBatHi
	}

	status, _ := d["battery_status"].(string)
	icon := "🔋"
	if status == "Charging" {
		icon = "⚡"
	}

	return style.Render(fmt.Sprintf("%s%d%%", icon, pct))
}

func BlockHeader() Block {
	return Block{
		Key: "header",
		Render: func(d map[string]any) string {
			name := "unknown"
			if n, ok := d["device_name"].(string); ok && n != "" {
				name = n
			} else if id, ok := d["device_id"].(string); ok {
				name = id
			}

			line := statusDot(d) + " " + hdrName.Render(name)
			if up := formatUptime(d); up != "" {
				line += hdrDim.Render(" · " + up)
			}

			var tags []string
			if bat := formatBattery(d); bat != "" {
				tags = append(tags, bat)
			}
			if ssid, ok := d["wifi_ssid"].(string); ok && ssid != "" {
				tags = append(tags, hdrDim.Render("📶 "+ssid))
			}
			if w, ok := d["weather"].(string); ok && w != "" {
				tags = append(tags, hdrDim.Render("🌍 "+w))
			}

			for _, tag := range tags {
				line += hdrDim.Render(" · ") + tag
			}

			return line
		},
	}
}
