package hyprland

import (
	"context"
	"encoding/json"
	"os/exec"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/collector/linux/hyprland/utils"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "hyprland-window", Collect: activeWindow},
		Applies:  collector.OnDEWM("hyprland"),
	})
}

func hyprctl(ctx context.Context, cmd string) (map[string]any, error) {
	out, err := exec.CommandContext(ctx, "hyprctl", cmd, "-j").Output()
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func activeWindow(ctx context.Context, snap presence.Snapshot) error {
	data, err := hyprctl(ctx, "activewindow")
	if err != nil {
		return err
	}
	if v, ok := data["title"].(string); ok {
		snap.Set(presence.KeyActiveWindow, v)
	}
	if v, ok := data["class"].(string); ok {
		snap.Set(presence.KeyActiveApp, utils.CleanAppName(v))
	}
	return nil
}
