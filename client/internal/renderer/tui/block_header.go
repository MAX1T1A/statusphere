package tui

import (
	"fmt"
	"strings"
	"time"

	"statusphere-client/internal/presence"

	"github.com/charmbracelet/lipgloss"
)

var (
	onlineDot   = lipgloss.NewStyle().Foreground(cOnline)
	idleDot     = lipgloss.NewStyle().Foreground(cIdle)
	offlineDot  = lipgloss.NewStyle().Foreground(cOffline)
	accountName = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	deviceName  = lipgloss.NewStyle().Bold(true).Foreground(cName)
	uptimeDim   = lipgloss.NewStyle().Foreground(cDim)
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

func deviceMeta(d presence.Snapshot) []string {
	var meta []string
	if up := formatUptime(d); up != "" {
		meta = append(meta, up)
	}
	if cpu, ok := d.Float(presence.KeyCPUPercent); ok {
		meta = append(meta, fmt.Sprintf("cpu %.0f%%", cpu))
	}
	if used, ok := d.Float(presence.KeyMemUsedMB); ok {
		if total, ok := d.Float(presence.KeyMemTotalMB); ok && total > 0 {
			meta = append(meta, fmt.Sprintf("mem %.1f/%.1fG", used/1024, total/1024))
		}
	}
	if load, ok := d.Float(presence.KeyLoadAvg1m); ok {
		meta = append(meta, fmt.Sprintf("load %.2f", load))
	}
	if pkgs, ok := d.Int(presence.KeyPackageCount); ok {
		meta = append(meta, fmt.Sprintf("%d pkgs", pkgs))
	}
	return meta
}

func deviceLine(d presence.Snapshot) string {
	name := "unknown"
	if n := d.DeviceName(); n != "" {
		name = n
	} else if id := d.DeviceID(); id != "" {
		name = id
	}
	line := statusDot(d) + " " + deviceName.Render(name)
	if meta := deviceMeta(d); len(meta) > 0 {
		line += uptimeDim.Render(" · " + strings.Join(meta, " · "))
	}
	return line
}

func groupHeader(g deviceGroup) string {
	lines := make([]string, 0, len(g.devices)+1)
	head := g.devices[0]
	if name := head.String(presence.KeyAccountName); name != "" {
		lines = append(lines, accountName.Render(name))
	}
	if head.Has(presence.KeyOffline) {
		lines = append(lines, offlineDot.Render("○")+uptimeDim.Render(" offline"))
		return strings.Join(lines, "\n")
	}
	for _, d := range g.devices {
		lines = append(lines, deviceLine(d))
	}
	return strings.Join(lines, "\n")
}
