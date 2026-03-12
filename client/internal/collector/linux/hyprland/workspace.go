package hyprland

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"statusphere-client/internal/models"
)

func ActiveWorkspace() func(models.Snapshot) {
	return func(snap models.Snapshot) {
		out, err := exec.Command("hyprctl", "activeworkspace", "-j").Output()
		if err != nil {
			return
		}
		var data map[string]any
		if err := json.Unmarshal(out, &data); err != nil {
			return
		}
		if v, ok := data["id"]; ok {
			snap["active_workspace"] = fmt.Sprint(v)
		}
	}
}
