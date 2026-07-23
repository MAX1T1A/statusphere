package tui

import (
	"fmt"
	"strings"
	"time"

	"statusphere-client/internal/presence"

	"github.com/charmbracelet/lipgloss"
)

var (
	onlineDot  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	idleDot    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	offlineDot = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	deviceName = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	uptimeDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	sysDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func statusDot(d presence.Snapshot) string {
	ts, ok := d.Int(presence.KeyLastSeen)
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

func formatUptime(d presence.Snapshot) string {
	v, ok := d.Float(presence.KeyUptimeHours)
	if !ok {
		return ""
	}
	switch {
	case v < 1:
		return fmt.Sprintf("%.0fm", v*60)
	case v < 24:
		return fmt.Sprintf("%.1fh", v)
	default:
		return fmt.Sprintf("%.0fd", v/24)
	}
}

func systemLine(d presence.Snapshot) string {
	var parts []string
	if cpu, ok := d.Float(presence.KeyCPUPercent); ok {
		parts = append(parts, fmt.Sprintf("cpu %.0f%%", cpu))
	}
	if used, ok := d.Float(presence.KeyMemUsedMB); ok {
		if total, ok := d.Float(presence.KeyMemTotalMB); ok && total > 0 {
			parts = append(parts, fmt.Sprintf("mem %.1f/%.1fG", used/1024, total/1024))
		}
	}
	if load, ok := d.Float(presence.KeyLoadAvg1m); ok {
		parts = append(parts, fmt.Sprintf("load %.2f", load))
	}
	if pkgs, ok := d.Int(presence.KeyPackageCount); ok {
		parts = append(parts, fmt.Sprintf("%d pkgs", pkgs))
	}
	return strings.Join(parts, " · ")
}

func BlockHeader() Block {
	return Block{
		Render: func(d presence.Snapshot) string {
			name := "unknown"
			if n := d.DeviceName(); n != "" {
				name = n
			} else if id := d.DeviceID(); id != "" {
				name = id
			}

			line := statusDot(d) + " " + deviceName.Render(name)
			if up := formatUptime(d); up != "" {
				line += uptimeDim.Render(" · " + up)
			}
			if sys := systemLine(d); sys != "" {
				line += "\n" + sysDim.Render(sys)
			}
			return line
		},
	}
}
