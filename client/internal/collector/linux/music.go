package linux

import (
	"context"
	"os/exec"
	"strings"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "music", Collect: music},
		Applies:  collector.OnOS("linux"),
	})
}

func music(ctx context.Context, snap presence.Snapshot) error {
	out, err := exec.CommandContext(ctx, "playerctl", "metadata", "--format", "{{artist}} - {{title}}").Output()
	if err != nil {
		return nil
	}
	track := strings.TrimSpace(string(out))
	if track != "" && track != "-" {
		snap.Set("music", track)
	}
	return nil
}
