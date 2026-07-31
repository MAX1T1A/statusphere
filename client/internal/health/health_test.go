package health

import (
	"testing"

	"statusphere-client/internal/presence"
)

func TestHealthyMachineSaysNothing(t *testing.T) {
	snap := presence.Snapshot{
		presence.KeyCPUPercent:      12.0,
		presence.KeyMemUsedMB:       4000.0,
		presence.KeyMemTotalMB:      16000.0,
		presence.KeyDiskUsedPercent: 41.0,
		presence.KeyLoadAvg1m:       0.7,
		presence.KeyCPUCount:        8,
	}

	Default().Annotate(snap)

	if snap.Has(presence.KeyHealth) {
		t.Fatalf("healthy snapshot carries %s = %v", presence.KeyHealth, snap[presence.KeyHealth])
	}
}

func TestCritBeatsWarnAndNamesBoth(t *testing.T) {
	snap := presence.Snapshot{
		presence.KeyDiskUsedPercent: 97.0,
		presence.KeyMemUsedMB:       15000.0,
		presence.KeyMemTotalMB:      16000.0,
	}

	Default().Annotate(snap)

	if got := snap.String(presence.KeyHealth); got != LevelCrit {
		t.Fatalf("health = %q, want %q", got, LevelCrit)
	}
	if got := snap.String(presence.KeyHealthNote); got != "disk 97% · mem 94%" {
		t.Fatalf("note = %q", got)
	}
}

// Load only means something per core: 6 is a quiet 16-core box and a drowning
// dual-core one.
func TestLoadIsJudgedPerCore(t *testing.T) {
	busy := presence.Snapshot{presence.KeyLoadAvg1m: 6.0, presence.KeyCPUCount: 2}
	calm := presence.Snapshot{presence.KeyLoadAvg1m: 6.0, presence.KeyCPUCount: 16}

	if level, _ := Default().Evaluate(busy); level != LevelCrit {
		t.Fatalf("2 cores under load 6: level = %q, want %q", level, LevelCrit)
	}
	if level, _ := Default().Evaluate(calm); level != LevelOK {
		t.Fatalf("16 cores under load 6: level = %q, want %q", level, LevelOK)
	}
}

// A zero band is how a machine opts out of a metric it does not care about.
func TestZeroBandDisablesTheMetric(t *testing.T) {
	th := Default()
	th.DiskPercent = Band{}

	if level, _ := th.Evaluate(presence.Snapshot{presence.KeyDiskUsedPercent: 99.0}); level != LevelOK {
		t.Fatalf("level = %q, want %q", level, LevelOK)
	}
}
