// Package health turns raw system metrics into ok/warn/crit. The verdict is
// computed on the machine being watched, not by whoever is looking: what counts
// as "too full" is a property of that box, and every UI reading the room -
// terminal, status bar, someone else's client - then agrees on the answer.
package health

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

const FileName = "health.json"

const (
	LevelOK   = "ok"
	LevelWarn = "warn"
	LevelCrit = "crit"
)

// Band is where a metric stops being fine. Warn at or above Warn, crit at or
// above Crit; a zero band switches that metric off.
type Band struct {
	Warn float64 `json:"warn"`
	Crit float64 `json:"crit"`
}

type Thresholds struct {
	CPUPercent  Band `json:"cpu_percent"`
	MemPercent  Band `json:"memory_percent"`
	DiskPercent Band `json:"disk_percent"`
	LoadPerCore Band `json:"load_per_core"`
}

// Default leans on load rather than on instant cpu: a single sample of
// /proc/stat spikes on any busy second, load average does not.
func Default() Thresholds {
	return Thresholds{
		CPUPercent:  Band{Warn: 95, Crit: 99},
		MemPercent:  Band{Warn: 90, Crit: 97},
		DiskPercent: Band{Warn: 85, Crit: 95},
		LoadPerCore: Band{Warn: 1.5, Crit: 3},
	}
}

func Load() Thresholds {
	data, err := config.Read(FileName)
	if err != nil {
		return Default()
	}
	t := Default()
	if err := json.Unmarshal(data, &t); err != nil {
		return Default()
	}
	return t
}

func EnsureConfig() {
	if _, err := config.Read(FileName); err == nil {
		return
	}
	if data, err := json.MarshalIndent(Default(), "", "  "); err == nil {
		_ = config.Write(FileName, append(data, '\n'), 0o600)
	}
}

type finding struct {
	level string
	text  string
}

// Annotate adds the verdict to a snapshot in place and returns it. A healthy
// machine gets no keys at all: silence is the normal state, and the room only
// carries the exceptions.
func (t Thresholds) Annotate(snap presence.Snapshot) presence.Snapshot {
	level, note := t.Evaluate(snap)
	if level == LevelOK {
		return snap
	}
	snap.Set(presence.KeyHealth, level)
	if note != "" {
		snap.Set(presence.KeyHealthNote, note)
	}
	return snap
}

func (t Thresholds) Evaluate(snap presence.Snapshot) (level, note string) {
	var findings []finding

	if cpu, ok := snap.Float(presence.KeyCPUPercent); ok {
		findings = append(findings, t.CPUPercent.check(cpu, fmt.Sprintf("cpu %.0f%%", cpu)))
	}
	if used, ok := snap.Float(presence.KeyMemUsedMB); ok {
		if total, ok := snap.Float(presence.KeyMemTotalMB); ok && total > 0 {
			pct := used / total * 100
			findings = append(findings, t.MemPercent.check(pct, fmt.Sprintf("mem %.0f%%", pct)))
		}
	}
	if disk, ok := snap.Float(presence.KeyDiskUsedPercent); ok {
		findings = append(findings, t.DiskPercent.check(disk, fmt.Sprintf("disk %.0f%%", disk)))
	}
	if load, ok := snap.Float(presence.KeyLoadAvg1m); ok {
		cores, _ := snap.Float(presence.KeyCPUCount)
		if cores < 1 {
			cores = 1
		}
		findings = append(findings, t.LoadPerCore.check(load/cores, fmt.Sprintf("load %.2f", load)))
	}

	return summarize(findings)
}

func (b Band) check(value float64, text string) finding {
	switch {
	case b.Crit > 0 && value >= b.Crit:
		return finding{level: LevelCrit, text: text}
	case b.Warn > 0 && value >= b.Warn:
		return finding{level: LevelWarn, text: text}
	default:
		return finding{level: LevelOK}
	}
}

// summarize keeps the worst level and names what earned it, crit first - the
// note has to fit on one line of a card, so it lists the reasons that matter.
func summarize(findings []finding) (string, string) {
	rank := map[string]int{LevelOK: 0, LevelWarn: 1, LevelCrit: 2}

	worst := LevelOK
	var reasons []finding
	for _, f := range findings {
		if f.level == LevelOK {
			continue
		}
		reasons = append(reasons, f)
		if rank[f.level] > rank[worst] {
			worst = f.level
		}
	}
	if worst == LevelOK {
		return LevelOK, ""
	}

	sort.SliceStable(reasons, func(i, j int) bool {
		return rank[reasons[i].level] > rank[reasons[j].level]
	})

	texts := make([]string, 0, len(reasons))
	for _, r := range reasons {
		texts = append(texts, r.text)
	}
	return worst, strings.Join(texts, " · ")
}
