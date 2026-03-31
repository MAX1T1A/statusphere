package custom

import (
	"bytes"
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

// load parses custom.json and returns both the config map and the keys in
// their original JSON order (Go maps do not preserve insertion order).
func load() (map[string]fieldConfig, []string) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, nil
	}

	var order []string
	configs := make(map[string]fieldConfig)

	dec := json.NewDecoder(bytes.NewReader(data))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, nil
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		key, ok := t.(string)
		if !ok {
			break
		}
		var cfg fieldConfig
		if err := dec.Decode(&cfg); err != nil {
			break
		}
		order = append(order, key)
		configs[key] = cfg
	}
	return configs, order
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
	fields, order := load()
	if len(fields) == 0 {
		return nil
	}

	var providers []func(models.Snapshot)
	for _, key := range order {
		cfg := fields[key]
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
	_, order := load()
	return order
}

func MergeKeys(raw []any) {
	current, _ := load()
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
