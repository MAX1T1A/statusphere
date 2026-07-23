package feed

import (
	"testing"
	"time"

	"statusphere-client/internal/presence"
)

func TestUpdateRequiresDeviceID(t *testing.T) {
	f := New()
	f.Update(presence.Snapshot{"foo": "bar"})
	if len(f.Snapshot()) != 0 {
		t.Fatal("snapshot without device_id must be ignored")
	}
	f.Update(presence.Snapshot{presence.KeyDeviceID: "dev1"})
	if len(f.Snapshot()) != 1 {
		t.Fatal("snapshot with device_id must be stored")
	}
}

func TestSnapshotStampsLastSeen(t *testing.T) {
	f := New()
	f.Update(presence.Snapshot{presence.KeyDeviceID: "dev1"})
	snaps := f.Snapshot()
	if len(snaps) != 1 {
		t.Fatalf("want 1 device, got %d", len(snaps))
	}
	if _, ok := snaps[0].Int(presence.KeyLastSeen); !ok {
		t.Fatal("last_seen must be stamped")
	}
}

func TestSnapshotPrunesStale(t *testing.T) {
	f := New()
	f.devices["old"] = &Device{
		Data:     presence.Snapshot{presence.KeyDeviceID: "old"},
		LastSeen: time.Now().Add(-2 * staleTTL),
	}
	f.Update(presence.Snapshot{presence.KeyDeviceID: "fresh"})

	snaps := f.Snapshot()
	if len(snaps) != 1 || snaps[0].DeviceID() != "fresh" {
		t.Fatalf("stale device should be pruned, got %v", snaps)
	}
	if _, ok := f.devices["old"]; ok {
		t.Fatal("stale device should be deleted from map")
	}
}
