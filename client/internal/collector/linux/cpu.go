package linux

import (
	"context"
	"math"
	"os"
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

func readCPUStat() (idle, total int64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, 0, nil
	}
	fields = fields[1:]

	var sum int64
	for _, f := range fields {
		v, _ := strconv.ParseInt(f, 10, 64)
		sum += v
	}

	idleVal, _ := strconv.ParseInt(fields[3], 10, 64)
	return idleVal, sum, nil
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
	return nil
}
