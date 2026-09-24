package layout

import (
	"os"
	"strings"
	"testing"
	"time"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

func write(t *testing.T, data string) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Write(FileName, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return config.File(FileName)
}

func TestMissingFileAddsNoKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	snap := (&Store{}).Annotate(presence.New())

	if snap.Has(presence.KeyLayout) {
		t.Fatalf("snapshot carries %s for a missing file", presence.KeyLayout)
	}
}

func TestValidFileIsPassedThroughAsIs(t *testing.T) {
	write(t, `{"updated_at":"2026-09-24T00:00:00Z","row":[{"kind":"clock"}],"detail":[]}`)

	snap := (&Store{}).Annotate(presence.New())

	got, ok := snap[presence.KeyLayout].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want a map", presence.KeyLayout, snap[presence.KeyLayout])
	}
	if got["updated_at"] != "2026-09-24T00:00:00Z" {
		t.Fatalf("updated_at = %v", got["updated_at"])
	}
	row, ok := got["row"].([]any)
	if !ok || len(row) != 1 {
		t.Fatalf("row = %#v", got["row"])
	}
}

func TestInvalidJSONAddsNoKey(t *testing.T) {
	write(t, `{not json`)

	snap := (&Store{}).Annotate(presence.New())

	if snap.Has(presence.KeyLayout) {
		t.Fatalf("snapshot carries %s for invalid JSON", presence.KeyLayout)
	}
}

func TestNonObjectAddsNoKey(t *testing.T) {
	write(t, `[1,2,3]`)

	snap := (&Store{}).Annotate(presence.New())

	if snap.Has(presence.KeyLayout) {
		t.Fatalf("snapshot carries %s for a non-object JSON value", presence.KeyLayout)
	}
}

func TestOversizeFileAddsNoKey(t *testing.T) {
	pad := strings.Repeat("x", maxSize)
	write(t, `{"row":[],"detail":[],"pad":"`+pad+`"}`)

	snap := (&Store{}).Annotate(presence.New())

	if snap.Has(presence.KeyLayout) {
		t.Fatalf("snapshot carries %s for an oversize file", presence.KeyLayout)
	}
}

func TestChangedFileIsPickedUp(t *testing.T) {
	path := write(t, `{"row":[{"kind":"clock"}],"detail":[]}`)
	s := &Store{}

	snap := s.Annotate(presence.New())
	got := snap[presence.KeyLayout].(map[string]any)
	row := got["row"].([]any)
	if len(row) != 1 {
		t.Fatalf("row = %#v", got["row"])
	}

	// Force a distinct mtime: some filesystems only keep second resolution.
	future := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(path, []byte(`{"row":[],"detail":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}

	snap = s.Annotate(presence.New())
	got = snap[presence.KeyLayout].(map[string]any)
	row = got["row"].([]any)
	if len(row) != 0 {
		t.Fatalf("row after update = %#v, want empty", got["row"])
	}
}
