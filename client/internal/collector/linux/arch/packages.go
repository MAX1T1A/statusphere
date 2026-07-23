package arch

import (
	"context"
	"os/exec"
	"strings"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "arch-packages", Collect: packageCount},
		Applies:  collector.OnDistro("arch"),
	})
}

func packageCount(ctx context.Context, snap presence.Snapshot) error {
	out, err := exec.CommandContext(ctx, "pacman", "-Q").Output()
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil
	}
	snap.Set(presence.KeyPackageCount, len(strings.Split(trimmed, "\n")))
	return nil
}
