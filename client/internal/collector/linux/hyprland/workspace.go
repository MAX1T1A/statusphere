package hyprland

import (
	"context"
	"fmt"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "hyprland-workspace", Collect: activeWorkspace},
		Applies:  collector.OnDEWM("hyprland"),
	})
}

func activeWorkspace(ctx context.Context, snap presence.Snapshot) error {
	data, err := hyprctl(ctx, "activeworkspace")
	if err != nil {
		return err
	}
	if v, ok := data["id"]; ok {
		snap.Set(presence.KeyActiveWorkspace, fmt.Sprint(v))
	}
	return nil
}
