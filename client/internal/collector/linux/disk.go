package linux

import (
	"context"
	"math"
	"syscall"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/presence"
)

func init() {
	collector.Register(collector.Descriptor{
		Provider: collector.Provider{Name: "disk", Collect: disk},
		Applies:  collector.OnOS("linux"),
	})
}

// mountPoint is the root filesystem: the one that fills up and takes the whole
// machine with it. Anything else is a per-host question and belongs in custom.json.
const mountPoint = "/"

func disk(_ context.Context, snap presence.Snapshot) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &st); err != nil {
		return err
	}
	if st.Blocks == 0 {
		return nil
	}

	size := st.Blocks * uint64(st.Bsize)
	// Free space reserved for root is unusable by anything that matters here,
	// so report what a normal process would actually get: Bavail, not Bfree.
	avail := st.Bavail * uint64(st.Bsize)
	used := size - st.Bfree*uint64(st.Bsize)

	snap.Set(presence.KeyDiskUsedPercent, math.Round(float64(used)/float64(size)*1000)/10)
	snap.Set(presence.KeyDiskFreeGB, math.Round(float64(avail)/(1<<30)*10)/10)
	return nil
}
