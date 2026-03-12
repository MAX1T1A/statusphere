package detector

import (
	"os"
	"strings"
)

func detectDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}

	kv := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(val), "\"")
	}

	id := strings.ToLower(kv["ID"])

	switch id {
	case "arch", "manjaro", "endeavouros":
		return "arch"
	case "ubuntu", "debian", "linuxmint":
		return "debian"
	case "fedora", "rhel", "centos":
		return "fedora"
	default:
		return id
	}
}
