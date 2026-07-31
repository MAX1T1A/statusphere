package linux

import "testing"

func busyPercent(before, after string) float64 {
	idle1, total1 := parseCPULine(before)
	idle2, total2 := parseCPULine(after)
	return (1 - float64(idle2-idle1)/float64(total2-total1)) * 100
}

// A box that spends its time waiting on the disk is idle, not busy. Counting
// iowait as work is what used to park a quiet machine at ~80%.
func TestIowaitCountsAsIdle(t *testing.T) {
	const (
		before = "cpu  1000 100 500 10000 30000 10 10 0 0 0"
		after  = "cpu  1005 100 505 10090 30300 10 10 0 0 0"
	)

	got := busyPercent(before, after)
	if want := 2.5; got < want-0.01 || got > want+0.01 {
		t.Fatalf("busy = %.2f%%, want %.2f%%", got, want)
	}
}

// guest and guest_nice are already inside user and nice, so a guest-heavy host
// must not have them counted twice in the total.
func TestGuestTimeIsNotDoubleCounted(t *testing.T) {
	const (
		before = "cpu  1000 100 500 10000 0 10 10 0 900 0"
		after  = "cpu  1100 100 500 10100 0 10 10 0 990 0"
	)

	got := busyPercent(before, after)
	if want := 50.0; got < want-0.01 || got > want+0.01 {
		t.Fatalf("busy = %.2f%%, want %.2f%%", got, want)
	}
}

func TestShortLineIsIgnored(t *testing.T) {
	if idle, total := parseCPULine("cpu 1 2 3"); idle != 0 || total != 0 {
		t.Fatalf("parseCPULine = (%d, %d), want (0, 0)", idle, total)
	}
}
