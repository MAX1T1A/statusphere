package linux

import (
	"context"
	"math"
	"os"
	"strconv"
	"strings"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "memory", Collect: memory},
		Applies:  collector.OnOS("linux"),
	})
}

func parseMeminfo() (map[string]int64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(val))
		if len(fields) == 0 {
			continue
		}
		v, _ := strconv.ParseInt(fields[0], 10, 64)
		result[strings.TrimSpace(key)] = v
	}
	return result, nil
}

func memory(_ context.Context, snap presence.Snapshot) error {
	info, err := parseMeminfo()
	if err != nil {
		return err
	}
	total := info["MemTotal"]
	available := info["MemAvailable"]
	if total == 0 {
		return nil
	}
	snap.Set(presence.KeyMemUsedMB, math.Round(float64(total-available)/1024*10)/10)
	snap.Set(presence.KeyMemTotalMB, math.Round(float64(total)/1024*10)/10)
	return nil
}
