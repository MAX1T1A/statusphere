package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func block(n int, w int, tag string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = tag + strings.Repeat("-", max(w-len(tag), 0))
	}
	return out
}

func TestPackFillsRightBeforeWrapping(t *testing.T) {
	out := packColumns([][]string{block(2, 10, "A"), block(2, 10, "B"), block(2, 10, "C")}, 40, 4)
	if len(out) != 2 {
		t.Fatalf("three narrow blocks should share one row, got %d lines:\n%s", len(out), strings.Join(out, "\n"))
	}
	for _, tag := range []string{"A", "B", "C"} {
		if !strings.Contains(out[0], tag) {
			t.Fatalf("%s missing from the first row: %q", tag, out[0])
		}
	}
}

func TestPackStacksUnderAShortColumn(t *testing.T) {
	tall := block(10, 20, "COVER")
	short := block(2, 20, "FACTS")
	extra := block(4, 20, "LYRICS")

	out := packColumns([][]string{tall, short, extra}, 48, 4)
	joined := strings.Join(out, "\n")
	if len(out) > 10 {
		t.Fatalf("the short block's spare room should absorb the next one, got %d lines:\n%s", len(out), joined)
	}
	if !strings.Contains(joined, "LYRICS") || !strings.Contains(joined, "FACTS") {
		t.Fatalf("both blocks should be placed:\n%s", joined)
	}
	for i, l := range out {
		if i < 10 && !strings.Contains(l, "COVER") {
			t.Fatalf("line %d should still carry the tall block: %q", i, l)
		}
	}
}

func TestPackStacksWhenTooWideToShare(t *testing.T) {
	out := packColumns([][]string{block(3, 30, "A"), block(3, 30, "B")}, 40, 4)
	joined := strings.Join(out, "\n")
	if len(out) < 6 {
		t.Fatalf("blocks too wide to share a row must stack:\n%s", joined)
	}
	if strings.Contains(out[0], "B") {
		t.Fatalf("B should not sit beside A: %q", out[0])
	}
	for _, l := range out {
		if ansi.StringWidth(l) > 40 {
			t.Fatalf("stacked output overflows: %q", l)
		}
	}
}

func TestPackBalancesColumnHeights(t *testing.T) {
	cover := block(10, 20, "COVER")
	facts := block(2, 26, "FACTS")
	weekly := block(4, 16, "WEEK")
	tops := block(4, 28, "TOPS")

	out := packColumns([][]string{cover, facts, weekly, tops}, 69, 4)
	if len(out) > 14 {
		t.Fatalf("columns should even out instead of running down one side (%d lines):\n%s", len(out), strings.Join(out, "\n"))
	}
	for _, tag := range []string{"COVER", "FACTS", "WEEK", "TOPS"} {
		if !strings.Contains(strings.Join(out, "\n"), tag) {
			t.Fatalf("%s went missing", tag)
		}
	}
	if !strings.Contains(out[0], "COVER") || !strings.Contains(out[0], "FACTS") {
		t.Fatalf("the first row should carry two columns: %q", out[0])
	}
}

func TestPackKeepsWithinWidth(t *testing.T) {
	out := packColumns([][]string{block(4, 18, "A"), block(2, 18, "B"), block(3, 18, "C"), block(6, 18, "D")}, 44, 4)
	for i, l := range out {
		if w := ansi.StringWidth(l); w > 44 {
			t.Fatalf("line %d overflows: %d > 44 (%q)", i, w, l)
		}
	}
}

func TestPackIgnoresEmptyBlocks(t *testing.T) {
	out := packColumns([][]string{nil, block(1, 5, "A"), {}}, 30, 4)
	if len(out) != 1 || !strings.Contains(out[0], "A") {
		t.Fatalf("empty blocks should be skipped: %v", out)
	}
}
