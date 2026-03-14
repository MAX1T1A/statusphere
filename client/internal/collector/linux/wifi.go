package linux

import (
	"os/exec"
	"statusphere-client/internal/models"
	"strings"
)

func WiFi() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		out, err := exec.Command("iwgetid", "-r").Output()
		if err == nil {
			if ssid := strings.TrimSpace(string(out)); ssid != "" {
				snap["wifi_ssid"] = ssid
				return
			}
		}

		out, err = exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi").Output()
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "yes:") {
				ssid := strings.TrimPrefix(line, "yes:")
				if ssid != "" {
					snap["wifi_ssid"] = ssid
				}
				return
			}
		}
	}
}
