package linux

import (
	"os"
	"path/filepath"
	"statusphere-client/internal/models"
	"strconv"
	"strings"
)

func Battery() func(models.Snapshot) {
	var batPath string

	return func(snap models.Snapshot) {
		if batPath == "" {
			entries, err := filepath.Glob("/sys/class/power_supply/BAT*")
			if err != nil || len(entries) == 0 {
				return
			}
			batPath = entries[0]
		}

		capData, err := os.ReadFile(filepath.Join(batPath, "capacity"))
		if err != nil {
			return
		}
		pct, err := strconv.Atoi(strings.TrimSpace(string(capData)))
		if err != nil {
			return
		}
		snap["battery_pct"] = pct

		statusData, _ := os.ReadFile(filepath.Join(batPath, "status"))
		snap["battery_status"] = strings.TrimSpace(string(statusData))
	}
}
