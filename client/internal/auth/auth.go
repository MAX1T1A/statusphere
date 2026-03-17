package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	RoomID    string `json:"room_id"`
	DeviceID  string `json:"device_id"`
}

func ConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "statusphere", "config.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Token == "" || cfg.ServerURL == "" {
		return nil, fmt.Errorf("incomplete config")
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	path := ConfigPath()
	os.MkdirAll(filepath.Dir(path), 0o700)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

type registerResponse struct {
	Token    string `json:"token"`
	RoomID   string `json:"room_id"`
	DeviceID string `json:"device_id"`
}

func Register(serverURL, roomID string) (*Config, error) {
	serverURL = strings.TrimRight(serverURL, "/")

	body, _ := json.Marshal(map[string]string{"room_id": roomID})
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Post(
		serverURL+"/auth/register",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("register failed: status %d", resp.StatusCode)
	}

	var result registerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("bad response: %w", err)
	}

	cfg := &Config{
		ServerURL: serverURL,
		Token:     result.Token,
		RoomID:    result.RoomID,
		DeviceID:  result.DeviceID,
	}
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return cfg, nil
}
