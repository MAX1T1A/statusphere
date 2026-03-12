package linux

import (
	"os"
	"strconv"
	"strings"

	"statusphere-client/internal/models"
)

func LoadAvg() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		data, err := os.ReadFile("/proc/loadavg")
		if err != nil {
			return
		}
		fields := strings.Fields(string(data))
		if len(fields) == 0 {
			return
		}
		v, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return
		}
		snap["load_avg_1m"] = v
	}
}
