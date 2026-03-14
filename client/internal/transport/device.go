package transport

import (
	"os"

	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
)

func ID() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return generateID()
	}

	path := filepath.Join(configDir, "statusphere", "device_id")

	if data, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id
		}
	}

	id := generateID()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(id), 0o644)
	return id
}

func Name() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		hostname, _ := os.Hostname()
		return hostname
	}

	path := filepath.Join(configDir, "statusphere", "device_name")

	if data, err := os.ReadFile(path); err == nil {
		if name := strings.TrimSpace(string(data)); name != "" {
			return name
		}
	}

	hostname, _ := os.Hostname()
	return hostname
}

func SetName(name string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}

	dir := filepath.Join(configDir, "statusphere")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "device_name"), []byte(name), 0o644)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
