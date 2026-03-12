package linux

import (
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"statusphere-client/internal/models"
)

func readCPUStat() (idle, total int64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)[1:]

	var sum int64
	for _, f := range fields {
		v, _ := strconv.ParseInt(f, 10, 64)
		sum += v
	}

	idleVal, _ := strconv.ParseInt(fields[3], 10, 64)
	return idleVal, sum, nil
}

func CPUPercent() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		idle1, total1, err := readCPUStat()
		if err != nil {
			return
		}

		time.Sleep(100 * time.Millisecond)

		idle2, total2, err := readCPUStat()
		if err != nil {
			return
		}

		totalDiff := float64(total2 - total1)
		if totalDiff == 0 {
			return
		}

		idleDiff := float64(idle2 - idle1)
		pct := math.Round((1-idleDiff/totalDiff)*1000) / 10
		snap["cpu_percent"] = pct
	}
}
