package linux

import (
	"context"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "cpu", Collect: cpuPercent},
		Applies:  collector.OnOS("linux"),
	})
}

// countedFields is user, nice, system, idle, iowait, irq, softirq, steal. The
// two that follow - guest and guest_nice - are already counted inside user and
// nice, so adding them would inflate the total and understate the load.
const countedFields = 8

func readCPUStat() (idle, total int64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	idle, total = parseCPULine(strings.SplitN(string(data), "\n", 2)[0])
	return idle, total, nil
}

func parseCPULine(line string) (idle, total int64) {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return 0, 0
	}
	fields = fields[1:]
	if len(fields) > countedFields {
		fields = fields[:countedFields]
	}

	var sum int64
	for _, f := range fields {
		v, _ := strconv.ParseInt(f, 10, 64)
		sum += v
	}

	// Idle is idle plus iowait: a core blocked on the disk is not doing work,
	// and counting it as busy pins a healthy box at 80% forever.
	idleVal, _ := strconv.ParseInt(fields[3], 10, 64)
	iowait, _ := strconv.ParseInt(fields[4], 10, 64)
	return idleVal + iowait, sum
}

func cpuPercent(ctx context.Context, snap presence.Snapshot) error {
	idle1, total1, err := readCPUStat()
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}

	idle2, total2, err := readCPUStat()
	if err != nil {
		return err
	}

	totalDiff := float64(total2 - total1)
	if totalDiff == 0 {
		return nil
	}

	idleDiff := float64(idle2 - idle1)
	snap.Set(presence.KeyCPUPercent, math.Round((1-idleDiff/totalDiff)*1000)/10)
	// Load average only means something next to the core count, and a card that
	// shows it has no other way to know how many cores are behind it.
	snap.Set(presence.KeyCPUCount, runtime.NumCPU())
	return nil
}
