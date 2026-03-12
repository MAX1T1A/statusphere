package transport

import (
	"net"
	"os"

	"crypto/sha256"
	"encoding/hex"
)

func ID() string {
	if v := os.Getenv("DEVICE_ID"); v != "" {
		return v
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	for _, iface := range ifaces {
		if len(iface.HardwareAddr) > 0 {
			h := sha256.Sum256(iface.HardwareAddr)
			return hex.EncodeToString(h[:8])
		}
	}

	return "unknown"
}
