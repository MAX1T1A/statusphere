package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

const fileName = "custom.json"

type fieldConfig struct {
	Cmd           string `json:"cmd"`
	RepeatSeconds int    `json:"repeat_seconds"`
}

type Manager struct {
	mu     sync.Mutex
	order  []string
	fields map[string]fieldConfig
}

func Load() *Manager {
	m := &Manager{fields: make(map[string]fieldConfig)}
	m.reload()
	return m
}

func (m *Manager) reload() {
	fields, order := parse()
	m.mu.Lock()
	m.fields = fields
	m.order = order
	m.mu.Unlock()
}

func parse() (map[string]fieldConfig, []string) {
	data, err := config.Read(fileName)
	if err != nil {
		return make(map[string]fieldConfig), nil
	}

	var order []string
	fields := make(map[string]fieldConfig)

	dec := json.NewDecoder(bytes.NewReader(data))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return fields, nil
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
		fields[key] = cfg
	}
	return fields, order
}

func (m *Manager) FieldNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.order...)
}

func (m *Manager) Providers() []collector.Provider {
	m.mu.Lock()
	order := append([]string(nil), m.order...)
	fields := m.fields
	m.mu.Unlock()

	var providers []collector.Provider
	for _, key := range order {
		cfg := fields[key]
		if cfg.Cmd == "" {
			continue
		}
		cache := &cachedResult{ttl: time.Duration(cfg.RepeatSeconds) * time.Second}
		k, cmd := key, cfg.Cmd
		providers = append(providers, collector.Provider{
			Name: "custom:" + k,
			Collect: func(ctx context.Context, snap presence.Snapshot) error {
				if val := cache.get(ctx, cmd); val != "" {
					snap.Set(k, val)
				}
				return nil
			},
		})
	}
	return providers
}

func (m *Manager) FieldsProvider() collector.Provider {
	return collector.Provider{
		Name: "custom-fields",
		Collect: func(_ context.Context, snap presence.Snapshot) error {
			if names := m.FieldNames(); len(names) > 0 {
				snap.Set(presence.KeyCustomFields, names)
			}
			return nil
		},
	}
}

func (m *Manager) MergeKeys(keys []string) {
	m.mu.Lock()
	changed := false
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, exists := m.fields[key]; !exists {
			m.fields[key] = fieldConfig{}
			m.order = append(m.order, key)
			changed = true
		}
	}
	if !changed {
		m.mu.Unlock()
		return
	}
	snapshot := make(map[string]fieldConfig, len(m.fields))
	for k, v := range m.fields {
		snapshot[k] = v
	}
	m.mu.Unlock()

	if data, err := json.MarshalIndent(snapshot, "", "  "); err == nil {
		_ = config.Write(fileName, data, 0o600)
	}
}

func EnsureConfig() {
	if _, err := config.Read(fileName); err == nil {
		return
	}
	data, _ := json.MarshalIndent(map[string]fieldConfig{}, "", "  ")
	_ = config.Write(fileName, data, 0o600)
}

type cachedResult struct {
	mu    sync.Mutex
	value string
	at    time.Time
	ttl   time.Duration
}

func (c *cachedResult) get(ctx context.Context, cmd string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl > 0 && !c.at.IsZero() && time.Since(c.at) < c.ttl && c.value != "" {
		return c.value
	}

	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
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
