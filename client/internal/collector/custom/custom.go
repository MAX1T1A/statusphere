package custom

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"statusphere-client/internal/models"
)

func configPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "statusphere", "custom.json")
}

func load() map[string]string {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil
	}
	var fields map[string]string
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil
	}
	return fields
}

func Providers() []func(models.Snapshot) {
	fields := load()
	if len(fields) == 0 {
		return nil
	}

	var providers []func(models.Snapshot)
	for key, cmd := range fields {
		k, c := key, cmd
		if c == "" {
			continue
		}
		providers = append(providers, func(snap models.Snapshot) {
			out, err := exec.Command("sh", "-c", c).Output()
			if err != nil {
				return
			}
			val := strings.TrimSpace(string(out))
			if val != "" {
				snap[k] = val
			}
		})
	}
	return providers
}

func FieldNames() []string {
	fields := load()
	if len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	return names
}

func MergeKeys(raw []any) {
	current := load()
	if current == nil {
		current = make(map[string]string)
	}

	changed := false
	for _, v := range raw {
		key, ok := v.(string)
		if !ok || key == "" {
			continue
		}
		if _, exists := current[key]; !exists {
			current[key] = ""
			changed = true
		}
	}

	if !changed {
		return
	}

	path := configPath()
	data, _ := json.MarshalIndent(current, "", "  ")
	_ = os.WriteFile(path, data, 0o600)
}

func EnsureConfig() {
	path := configPath()
	if _, err := os.Stat(path); err == nil {
		return
	}
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o700)
	data, _ := json.MarshalIndent(map[string]string{}, "", "  ")
	_ = os.WriteFile(path, data, 0o600)
}
