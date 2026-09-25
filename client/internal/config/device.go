package transport

import (
	"os"
	"strings"

	"statusphere-client/internal/config"
)

const nameFile = "device_name"

func loadName() string {
	if data, err := config.Read(nameFile); err == nil {
		if name := strings.TrimSpace(string(data)); name != "" {
			return name
		}
	}
	hostname, _ := os.Hostname()
	return hostname
}

func saveName(name string) {
	_ = config.Write(nameFile, []byte(name), 0o600)
}
