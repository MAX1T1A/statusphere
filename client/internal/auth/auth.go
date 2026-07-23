package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"statusphere-client/internal/config"
)

const fileName = "config.json"

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
}

func (c *Config) tokenPart(i int) string {
	parts := strings.Split(c.Token, ":")
	if len(parts) != 3 {
		return ""
	}
	return parts[i]
}

func (c *Config) RoomID() string   { return c.tokenPart(0) }
func (c *Config) DeviceID() string { return c.tokenPart(1) }

func ConfigPath() string {
	return config.File(fileName)
}

func Load() (*Config, error) {
	data, err := config.Read(fileName)
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
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return config.Write(fileName, data, 0o600)
}

type registerResponse struct {
	Token string `json:"token"`
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("register failed: status %d", resp.StatusCode)
	}

	var result registerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("bad response: %w", err)
	}

	cfg := &Config{ServerURL: serverURL, Token: result.Token}
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return cfg, nil
}
