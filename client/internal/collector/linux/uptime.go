package linux

import (
	"os"
	"statusphere-client/internal/models"
	"strconv"
	"strings"
)

func Uptime() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		data, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return
		}
		val, _ := strconv.ParseFloat(strings.Fields(string(data))[0], 64)
		snap["uptime_hours"] = val / 3600
	}
}
