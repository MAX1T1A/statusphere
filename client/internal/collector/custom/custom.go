package custom

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"statusphere-client/internal/models"
)

type fieldConfig struct {
	Cmd           string `json:"cmd"`
	RepeatSeconds int    `json:"repeat_seconds"`
}

func configPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "statusphere", "custom.json")
}

func load() map[string]fieldConfig {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil
	}
	var fields map[string]fieldConfig
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil
	}
	return fields
}

type cachedResult struct {
	mu    sync.Mutex
	value string
	at    time.Time
	ttl   time.Duration
}

func (c *cachedResult) get(cmd string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl > 0 && time.Since(c.at) < c.ttl && c.value != "" {
		return c.value
	}

	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return c.value
	}
	val := strings.TrimSpace(string(out))
	if val != "" {
		c.value = val
		c.at = time.Now()
	}
	return c.value
}

func Providers() []func(models.Snapshot) {
	fields := load()
	if len(fields) == 0 {
		return nil
	}

	var providers []func(models.Snapshot)
	for key, cfg := range fields {
		k, c := key, cfg
		if c.Cmd == "" {
			continue
		}
		cache := &cachedResult{
			ttl: time.Duration(c.RepeatSeconds) * time.Second,
		}
		providers = append(providers, func(snap models.Snapshot) {
			if val := cache.get(c.Cmd); val != "" {
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
		current = make(map[string]fieldConfig)
	}

	changed := false
	for _, v := range raw {
		key, ok := v.(string)
		if !ok || key == "" {
			continue
		}
		if _, exists := current[key]; !exists {
			current[key] = fieldConfig{}
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
	data, _ := json.MarshalIndent(map[string]fieldConfig{}, "", "  ")
	_ = os.WriteFile(path, data, 0o600)
}
