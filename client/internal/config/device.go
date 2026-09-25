package config

import (
	"os"
	"strings"
)

const deviceNameFile = "device_name"

func DeviceName() string {
	if data, err := Read(deviceNameFile); err == nil {
		if name := strings.TrimSpace(string(data)); name != "" {
			return name
		}
	}
	hostname, _ := os.Hostname()
	return hostname
}

func SetDeviceName(name string) error {
	return Write(deviceNameFile, []byte(name), 0o600)
}
