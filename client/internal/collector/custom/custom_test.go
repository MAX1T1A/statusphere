package custom

import (
	"context"
	"os"
	"testing"
	"time"

	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

func write(t *testing.T, data string) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Write(fileName, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return config.File(fileName)
}

func rewrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

func collect(t *testing.T, m *Manager) presence.Snapshot {
	t.Helper()
	snap := presence.New()
	for _, p := range m.Providers() {
		if err := p.Collect(context.Background(), snap); err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
	}
	if err := m.FieldsProvider().Collect(context.Background(), snap); err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestChangedFileAddsField(t *testing.T) {
	path := write(t, `{"weather":{"cmd":"echo sunny","repeat_seconds":60}}`)
	m := Load()

	snap := collect(t, m)
	if snap.String("weather") != "sunny" {
		t.Fatalf("weather = %q, want sunny", snap.String("weather"))
	}
	if got := snap.Strings(presence.KeyCustomFields); len(got) != 1 || got[0] != "weather" {
		t.Fatalf("custom_fields = %v", got)
	}

	rewrite(t, path, `{"weather":{"cmd":"echo sunny","repeat_seconds":60},"git":{"cmd":"echo dirty","repeat_seconds":60}}`)

	snap = collect(t, m)
	if snap.String("git") != "dirty" {
		t.Fatalf("git = %q, want dirty", snap.String("git"))
	}
	if got := snap.Strings(presence.KeyCustomFields); len(got) != 2 {
		t.Fatalf("custom_fields = %v, want 2 fields", got)
	}
}

func TestChangedCmdGivesNewValueOnNextCollect(t *testing.T) {
	path := write(t, `{"weather":{"cmd":"echo sunny","repeat_seconds":60}}`)
	m := Load()

	snap := collect(t, m)
	if snap.String("weather") != "sunny" {
		t.Fatalf("weather = %q, want sunny", snap.String("weather"))
	}

	rewrite(t, path, `{"weather":{"cmd":"echo rainy","repeat_seconds":60}}`)

	snap = collect(t, m)
	if snap.String("weather") != "rainy" {
		t.Fatalf("weather after cmd change = %q, want rainy (stale cache not dropped)", snap.String("weather"))
	}
}

func TestRemovedFieldIsGone(t *testing.T) {
	path := write(t, `{"weather":{"cmd":"echo sunny","repeat_seconds":60},"git":{"cmd":"echo dirty","repeat_seconds":60}}`)
	m := Load()

	snap := collect(t, m)
	if !snap.Has("git") {
		t.Fatalf("git missing before removal: %v", snap)
	}

	rewrite(t, path, `{"weather":{"cmd":"echo sunny","repeat_seconds":60}}`)

	snap = collect(t, m)
	if snap.Has("git") {
		t.Fatalf("git = %v, want removed from the snapshot", snap["git"])
	}
	if got := snap.Strings(presence.KeyCustomFields); len(got) != 1 || got[0] != "weather" {
		t.Fatalf("custom_fields = %v, want only weather", got)
	}
}

func TestInvalidJSONKeepsPreviousFields(t *testing.T) {
	path := write(t, `{"weather":{"cmd":"echo sunny","repeat_seconds":60}}`)
	m := Load()

	snap := collect(t, m)
	if snap.String("weather") != "sunny" {
		t.Fatalf("weather = %q, want sunny", snap.String("weather"))
	}

	rewrite(t, path, `{not json`)

	snap = collect(t, m)
	if snap.String("weather") != "sunny" {
		t.Fatalf("weather after invalid JSON = %q, want sunny (last good set kept)", snap.String("weather"))
	}
	if got := snap.Strings(presence.KeyCustomFields); len(got) != 1 || got[0] != "weather" {
		t.Fatalf("custom_fields after invalid JSON = %v, want unchanged", got)
	}
}

func TestMergeKeysWriteIsNotReparsedAsAChange(t *testing.T) {
	write(t, `{}`)
	m := Load()

	m.MergeKeys([]string{"weather", "git"})

	if got := m.FieldNames(); len(got) != 2 || got[0] != "weather" || got[1] != "git" {
		t.Fatalf("FieldNames() = %v, want insertion order [weather git]", got)
	}

	collect(t, m)

	if got := m.FieldNames(); len(got) != 2 || got[0] != "weather" || got[1] != "git" {
		t.Fatalf("FieldNames() after collect = %v, want insertion order preserved", got)
	}
}
