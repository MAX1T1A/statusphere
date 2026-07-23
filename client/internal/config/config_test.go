package config_test

import (
	"path/filepath"
	"testing"

	"statusphere-client/internal/config"
)

func TestPathsUseXDGConfig(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)

	want := filepath.Join(base, config.AppName)
	if config.Dir() != want {
		t.Fatalf("Dir() = %q; want %q", config.Dir(), want)
	}
	if config.File("x.json") != filepath.Join(want, "x.json") {
		t.Fatalf("File() = %q", config.File("x.json"))
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := config.Write("device_name", []byte("laptop"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := config.Read("device_name")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "laptop" {
		t.Fatalf("round trip = %q; want laptop", data)
	}
}
