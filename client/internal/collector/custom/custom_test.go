package custom

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

// collect runs the providers, waits for any cache misses to resolve in the
// background, then collects again so the snapshot reflects settled values -
// a fresh field only appears once its first background run has completed.
func collect(t *testing.T, m *Manager) presence.Snapshot {
	t.Helper()
	collectOnce(t, m)
	waitForCaches(t, m)
	return collectOnce(t, m)
}

func collectOnce(t *testing.T, m *Manager) presence.Snapshot {
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

func waitForCaches(t *testing.T, m *Manager) {
	t.Helper()
	waitUntil(t, 2*time.Second, func() bool {
		_, caches := m.snapshotCaches()
		for _, c := range caches {
			if _, running, _ := snapshotCache(c); running {
				return false
			}
		}
		return true
	})
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

func withCommandTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	prev := commandTimeout
	commandTimeout = d
	t.Cleanup(func() { commandTimeout = prev })
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func snapshotCache(c *cachedResult) (value string, running bool, lastErr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value, c.running, c.lastErr
}

func TestGetDoesNotBlockOnSlowCommand(t *testing.T) {
	withCommandTimeout(t, 50*time.Millisecond)
	c := &cachedResult{field: "f", cmd: "sleep 5"}

	start := time.Now()
	val := c.get()
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Fatalf("get() took %v, want it to return without waiting for the command", elapsed)
	}
	if val != "" {
		t.Fatalf("get() = %q on first call, want empty until the background run completes", val)
	}

	// Drain the background run before the next test overrides commandTimeout
	// again, so it doesn't race with this goroutine's read of the var.
	waitUntil(t, 2*time.Second, func() bool {
		_, running, _ := snapshotCache(c)
		return !running
	})
}

func TestTimeoutKeepsLastGoodValue(t *testing.T) {
	withCommandTimeout(t, 100*time.Millisecond)
	marker := filepath.Join(t.TempDir(), "marker")
	cmd := fmt.Sprintf(`if [ -f %q ]; then sleep 5; else touch %q; echo ok; fi`, marker, marker)
	c := &cachedResult{field: "f", cmd: cmd}

	c.get()
	waitUntil(t, 2*time.Second, func() bool {
		val, running, _ := snapshotCache(c)
		return !running && val == "ok"
	})

	c.get()
	waitUntil(t, 2*time.Second, func() bool {
		_, running, _ := snapshotCache(c)
		return !running
	})

	val, _, lastErr := snapshotCache(c)
	if val != "ok" {
		t.Fatalf("value after timeout = %q, want last good value %q kept", val, "ok")
	}
	if lastErr == "" {
		t.Fatal("lastErr empty, want the timeout to be recorded")
	}
}

func TestNoOverlappingRuns(t *testing.T) {
	starts := filepath.Join(t.TempDir(), "starts")
	cmd := fmt.Sprintf(`echo start >> %q; sleep 0.3`, starts)
	c := &cachedResult{field: "f", cmd: cmd}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.get()
		}()
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)
	data, err := os.ReadFile(starts)
	if err != nil {
		t.Fatalf("reading starts file: %v", err)
	}
	if got := len(strings.Split(strings.TrimSpace(string(data)), "\n")); got != 1 {
		t.Fatalf("command started %d times concurrently, want exactly 1", got)
	}

	waitUntil(t, 2*time.Second, func() bool {
		_, running, _ := snapshotCache(c)
		return !running
	})
}

func TestChildProcessKilledOnTimeout(t *testing.T) {
	withCommandTimeout(t, 150*time.Millisecond)
	pidFile := filepath.Join(t.TempDir(), "pid")
	cmd := fmt.Sprintf(`sleep 5 & echo $! > %q; wait`, pidFile)
	c := &cachedResult{field: "f", cmd: cmd}

	c.get()
	waitUntil(t, 3*time.Second, func() bool {
		_, running, _ := snapshotCache(c)
		return !running
	})

	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("reading pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parsing pid: %v", err)
	}
	if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
		t.Fatalf("process %d still alive after timeout (err=%v), want it gone", pid, err)
	}
}
