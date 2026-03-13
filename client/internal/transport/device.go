package transport

import (
	"os"

	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
)

func ID() string {
	if v := os.Getenv("DEVICE_ID"); v != "" {
		return v
	}

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
	if v := os.Getenv("DEVICE_NAME"); v != "" {
		return v
	}
	hostname, _ := os.Hostname()
	return hostname
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
