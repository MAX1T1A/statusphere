package linux

import (
	"os/exec"
	"strings"

	"statusphere-client/internal/models"
)

func Music() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		out, err := exec.Command("playerctl", "metadata", "--format", "{{artist}} - {{title}}").Output()
		if err != nil {
			return
		}
		track := strings.TrimSpace(string(out))
		if track != "" && track != " - " {
			snap["music"] = track
		}
	}
}
