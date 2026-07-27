package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func overlay(base, box string, row, col int) string {
	if box == "" {
		return base
	}
	lines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")

	boxW := 0
	for _, l := range boxLines {
		if w := ansi.StringWidth(l); w > boxW {
			boxW = w
		}
	}
	if row < 0 {
		row = 0
	}
	if col < 0 {
		col = 0
	}
	if over := row + len(boxLines) - len(lines); over > 0 {
		row = max(row-over, 0)
	}

	for i, bl := range boxLines {
		y := row + i
		if y >= len(lines) {
			break
		}
		lines[y] = spliceLine(lines[y], bl, col, boxW)
	}
	return strings.Join(lines, "\n")
}

func spliceLine(base, box string, col, boxW int) string {
	baseW := ansi.StringWidth(base)

	left := ""
	if col > 0 {
		left = ansi.Cut(base, 0, min(col, baseW))
		if gap := col - baseW; gap > 0 {
			left += strings.Repeat(" ", gap)
		}
	}

	if w := ansi.StringWidth(box); w < boxW {
		box += strings.Repeat(" ", boxW-w)
	}

	right := ""
	if end := col + boxW; end < baseW {
		right = ansi.Cut(base, end, baseW)
	}
	return left + box + right
}
