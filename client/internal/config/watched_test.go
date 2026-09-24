package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"statusphere-client/internal/config"
)

func touch(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}

func TestWatchedFirstCallOnExistingFileIsChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	touch(t, path, "one")
	w := &config.Watched{Path: path}

	changed, err := w.Changed()
	if err != nil || !changed {
		t.Fatalf("Changed() = %v, %v; want true, nil", changed, err)
	}
	data, err := w.Read()
	if err != nil || string(data) != "one" {
		t.Fatalf("Read() = %q, %v; want %q, nil", data, err, "one")
	}
}

func TestWatchedUnchangedFileReportsNoChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	touch(t, path, "one")
	w := &config.Watched{Path: path}
	if _, err := w.Changed(); err != nil {
		t.Fatal(err)
	}

	changed, err := w.Changed()
	if err != nil || changed {
		t.Fatalf("Changed() on unchanged file = %v, %v; want false, nil", changed, err)
	}
}

func TestWatchedRewriteIsChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	touch(t, path, "one")
	w := &config.Watched{Path: path}
	if _, err := w.Changed(); err != nil {
		t.Fatal(err)
	}

	touch(t, path, "two")
	changed, err := w.Changed()
	if err != nil || !changed {
		t.Fatalf("Changed() after rewrite = %v, %v; want true, nil", changed, err)
	}
	data, err := w.Read()
	if err != nil || string(data) != "two" {
		t.Fatalf("Read() after rewrite = %q, %v; want %q, nil", data, err, "two")
	}
}

func TestWatchedMissingFileErrorsAndIsUnseen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	w := &config.Watched{Path: path}

	changed, err := w.Changed()
	if err == nil {
		t.Fatal("Changed() on a missing file: want a stat error")
	}
	if changed {
		t.Fatal("Changed() on a missing file that was never seen: want false")
	}
	if w.Seen() {
		t.Fatal("Seen() after a missing file: want false")
	}
}

func TestWatchedDisappearanceIsReportedAsChangedOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	touch(t, path, "one")
	w := &config.Watched{Path: path}
	if _, err := w.Changed(); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	changed, err := w.Changed()
	if err == nil {
		t.Fatal("Changed() after removal: want a stat error")
	}
	if !changed {
		t.Fatal("Changed() right after a file disappears: want true (transition out of existing)")
	}

	changed, err = w.Changed()
	if err == nil {
		t.Fatal("Changed() while still missing: want a stat error")
	}
	if changed {
		t.Fatal("Changed() while still missing on the second call: want false")
	}
}

func TestWatchedReadRefusesOversize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	touch(t, path, "0123456789")
	w := &config.Watched{Path: path, MaxSize: 5}

	if _, err := w.Changed(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Read(); !errors.Is(err, config.ErrTooLarge) {
		t.Fatalf("Read() over MaxSize = %v, want ErrTooLarge", err)
	}
}

func TestWatchedSyncAvoidsReportingOwnWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	touch(t, path, "one")
	w := &config.Watched{Path: path}
	if _, err := w.Changed(); err != nil {
		t.Fatal(err)
	}

	touch(t, path, "two")
	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}

	changed, err := w.Changed()
	if err != nil || changed {
		t.Fatalf("Changed() right after Sync = %v, %v; want false, nil", changed, err)
	}
}
