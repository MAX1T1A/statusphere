package linux

import (
	"context"
	"os"
	"strconv"
	"strings"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "uptime", Collect: uptime},
		Applies:  collector.OnOS("linux"),
	})
}

func uptime(_ context.Context, snap presence.Snapshot) error {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return nil
	}
	val, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return err
	}
	snap.Set(presence.KeyUptimeHours, val/3600)
	return nil
}
