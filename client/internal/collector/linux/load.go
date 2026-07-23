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
		Provider: collector.Provider{Name: "load", Collect: loadAvg},
		Applies:  collector.OnOS("linux"),
	})
}

func loadAvg(_ context.Context, snap presence.Snapshot) error {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return nil
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return err
	}
	snap.Set(presence.KeyLoadAvg1m, v)
	return nil
}
