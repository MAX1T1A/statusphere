package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"statusphere-client/internal/collector"
	"statusphere-client/internal/config"
	"statusphere-client/internal/presence"
)

const fileName = "custom.json"

const defaultCommandTimeout = 5 * time.Second

const commandKillGrace = 500 * time.Millisecond

var commandTimeout = defaultCommandTimeout

type fieldConfig struct {
	Cmd           string `json:"cmd"`
	RepeatSeconds int    `json:"repeat_seconds"`
}

type Manager struct {
	mu     sync.Mutex
	order  []string
	fields map[string]fieldConfig
	caches map[string]*cachedResult

	attempted bool
	mod       time.Time
	size      int64
	lastErr   string
}

func Load() *Manager {
	m := &Manager{fields: make(map[string]fieldConfig), caches: make(map[string]*cachedResult)}
	m.refresh()
	return m
}

func (m *Manager) refresh() {
	path := config.File(fileName)
	info, err := os.Stat(path)

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		if !m.attempted {
			m.fields = make(map[string]fieldConfig)
			m.order = nil
			m.caches = make(map[string]*cachedResult)
		}
		return
	}
	if m.attempted && info.ModTime().Equal(m.mod) && info.Size() == m.size {
		return
	}
	m.attempted = true
	m.mod = info.ModTime()
	m.size = info.Size()

	data, err := os.ReadFile(path)
	if err != nil {
		m.logReloadFailure(err)
		return
	}
	fields, order, err := parseData(data)
	if err != nil {
		m.logReloadFailure(err)
		return
	}
	m.lastErr = ""
	m.fields = fields
	m.order = order
	m.syncCachesLocked()
}

func (m *Manager) logReloadFailure(err error) {
	reason := err.Error()
	if reason == m.lastErr {
		return
	}
	m.lastErr = reason
	log.Printf("event=custom_fields_reload_failed reason=%q", reason)
}

func (m *Manager) syncCachesLocked() {
	next := make(map[string]*cachedResult, len(m.fields))
	for key, cfg := range m.fields {
		if cfg.Cmd == "" {
			continue
		}
		if c, ok := m.caches[key]; ok && c.cmd == cfg.Cmd {
			next[key] = c
			continue
		}
		next[key] = &cachedResult{field: key, cmd: cfg.Cmd, ttl: time.Duration(cfg.RepeatSeconds) * time.Second}
	}
	m.caches = next
}

func parseData(data []byte) (map[string]fieldConfig, []string, error) {
	var order []string
	fields := make(map[string]fieldConfig)

	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if t != json.Delim('{') {
		return nil, nil, fmt.Errorf("not a JSON object")
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key, ok := t.(string)
		if !ok {
			return nil, nil, fmt.Errorf("unexpected key token %v", t)
		}
		var cfg fieldConfig
		if err := dec.Decode(&cfg); err != nil {
			return nil, nil, err
		}
		order = append(order, key)
		fields[key] = cfg
	}
	if _, err := dec.Token(); err != nil {
		return nil, nil, err
	}
	return fields, order, nil
}

func (m *Manager) FieldNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.order...)
}

// snapshotCaches copies out the order and caches map reference so the caller
// can run cmd exec per field without holding the manager lock for the duration.
func (m *Manager) snapshotCaches() ([]string, map[string]*cachedResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.order...), m.caches
}

func (m *Manager) Providers() []collector.Provider {
	return []collector.Provider{{
		Name: "custom",
		Collect: func(_ context.Context, snap presence.Snapshot) error {
			m.refresh()
			order, caches := m.snapshotCaches()
			for _, key := range order {
				cache, ok := caches[key]
				if !ok {
					continue
				}
				if val := cache.get(); val != "" {
					snap.Set(key, val)
				}
			}
			return nil
		},
	}}
}

func (m *Manager) FieldsProvider() collector.Provider {
	return collector.Provider{
		Name: "custom-fields",
		Collect: func(_ context.Context, snap presence.Snapshot) error {
			m.refresh()
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

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return
	}
	if err := config.Write(fileName, data, 0o600); err != nil {
		return
	}

	// Without this, refresh() would see its own write as an external change
	// and re-parse the file, reordering fields into json.Marshal's
	// alphabetical key order instead of insertion order.
	if info, err := os.Stat(config.File(fileName)); err == nil {
		m.mu.Lock()
		m.attempted = true
		m.mod = info.ModTime()
		m.size = info.Size()
		m.mu.Unlock()
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
	mu      sync.Mutex
	field   string
	cmd     string
	value   string
	at      time.Time
	ttl     time.Duration
	running bool
	lastErr string
}

// get returns the last known value without running cmd itself: a stale or
// missing value triggers a background refresh (at most one in flight per
// field) so a slow or hanging command never blocks the collection tick.
func (c *cachedResult) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl > 0 && !c.at.IsZero() && time.Since(c.at) < c.ttl && c.value != "" {
		return c.value
	}
	if !c.running {
		c.running = true
		go c.refresh()
	}
	return c.value
}

func (c *cachedResult) refresh() {
	c.mu.Lock()
	cmd := c.cmd
	c.mu.Unlock()

	val, err := runCommand(cmd)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	if err != nil {
		c.logFailure(err)
		return
	}
	c.lastErr = ""
	if val != "" {
		c.value = val
		c.at = time.Now()
	}
}

func (c *cachedResult) logFailure(err error) {
	reason := err.Error()
	if reason == c.lastErr {
		return
	}
	c.lastErr = reason
	log.Printf("event=custom_field_cmd_failed field=%q reason=%q", c.field, reason)
}

// runCommand kills the whole process group on timeout, not just the
// immediate "sh" child, so a still-running grandchild (e.g. curl spawned by
// "sh -c") doesn't outlive the timeout as an orphan.
func runCommand(cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
	c.WaitDelay = commandKillGrace

	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
