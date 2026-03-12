package hyprland

import (
	"encoding/json"
	"os/exec"

	"statusphere-client/internal/collector/linux/hyprland/utils"
	"statusphere-client/internal/models"
)

func hyprctlActiveWindow() (map[string]any, error) {
	out, err := exec.Command("hyprctl", "activewindow", "-j").Output()
	if err != nil {
		return nil, err
	}
	var data map[string]any
	err = json.Unmarshal(out, &data)
	return data, err
}

func ActiveWindow() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		data, err := hyprctlActiveWindow()
		if err != nil {
			return
		}
		if v, ok := data["title"].(string); ok {
			snap["active_window"] = v
		}
		if v, ok := data["class"].(string); ok {
			snap["active_app"] = utils.CleanAppName(v)
		}
	}
}
