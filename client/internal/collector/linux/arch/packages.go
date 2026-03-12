package arch

import (
	"os/exec"
	"strings"

	"statusphere-client/internal/models"
)

func PackageCount() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		out, err := exec.Command("pacman", "-Q").Output()
		if err != nil {
			return
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		snap["package_count"] = len(lines)
	}
}
