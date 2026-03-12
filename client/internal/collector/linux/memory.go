package linux

import (
	"math"
	"os"
	"strconv"
	"strings"

	"statusphere-client/internal/models"
)

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

func Memory() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		info, err := parseMeminfo()
		if err != nil {
			return
		}
		total := info["MemTotal"]
		available := info["MemAvailable"]
		snap["memory_used_mb"] = math.Round(float64(total-available)/1024*10) / 10
		snap["memory_total_mb"] = math.Round(float64(total)/1024*10) / 10
	}
}
