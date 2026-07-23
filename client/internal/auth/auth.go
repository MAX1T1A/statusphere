package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"statusphere-client/internal/config"
)

const fileName = "config.json"

var client = &http.Client{Timeout: 10 * time.Second}

type Config struct {
	ServerURL     string `json:"server_url"`
	AccountSecret string `json:"account_secret,omitempty"`
	AccountID     string `json:"account_id"`
	DeviceID      string `json:"device_id"`
	Token         string `json:"token"`
	RoomID        string `json:"room_id"`
}

func ConfigPath() string { return config.File(fileName) }

func Load() (*Config, error) {
	data, err := config.Read(fileName)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Token == "" || cfg.ServerURL == "" || cfg.RoomID == "" {
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

func do(method, url, token string, body, out any) error {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Room-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: status %d", method, url, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Config) endpoint(path string) string {
	return strings.TrimRight(c.ServerURL, "/") + path
}

type accountResponse struct {
	AccountID string `json:"account_id"`
	DeviceID  string `json:"device_id"`
	RoomID    string `json:"room_id"`
	Token     string `json:"token"`
}

func newSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func deviceName() string {
	name, _ := os.Hostname()
	return name
}

func Register(serverURL string) (*Config, error) {
	serverURL = strings.TrimRight(serverURL, "/")
	secret := newSecret()

	var resp accountResponse
	if err := do(http.MethodPost, serverURL+"/accounts/register", "", map[string]string{"secret": secret}, &resp); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}

	cfg := &Config{
		ServerURL:     serverURL,
		AccountSecret: secret,
		AccountID:     resp.AccountID,
		DeviceID:      resp.DeviceID,
		Token:         resp.Token,
		RoomID:        resp.RoomID,
	}
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return cfg, nil
}

func LinkDevice(serverURL, code string) (*Config, error) {
	serverURL = strings.TrimRight(serverURL, "/")

	var resp accountResponse
	body := map[string]string{"code": code, "name": deviceName()}
	if err := do(http.MethodPost, serverURL+"/devices/link", "", body, &resp); err != nil {
		return nil, fmt.Errorf("link: %w", err)
	}

	cfg := &Config{
		ServerURL: serverURL,
		AccountID: resp.AccountID,
		DeviceID:  resp.DeviceID,
		Token:     resp.Token,
		RoomID:    resp.RoomID,
	}
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	return cfg, nil
}

func (c *Config) NewDeviceCode() (string, error) {
	var resp struct {
		Code string `json:"code"`
	}
	if err := do(http.MethodPost, c.endpoint("/devices/link-code"), c.Token, nil, &resp); err != nil {
		return "", err
	}
	return resp.Code, nil
}

func (c *Config) Invite() (string, error) {
	var resp struct {
		Code string `json:"code"`
	}
	if err := do(http.MethodPost, c.endpoint("/rooms/invite"), c.Token, nil, &resp); err != nil {
		return "", err
	}
	return resp.Code, nil
}

func (c *Config) Join(code string) error {
	var resp struct {
		RoomID string `json:"room_id"`
	}
	if err := do(http.MethodPost, c.endpoint("/rooms/join"), c.Token, map[string]string{"code": code}, &resp); err != nil {
		return err
	}
	c.RoomID = resp.RoomID
	return c.Save()
}

type DeviceInfo struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
	Revoked  bool   `json:"revoked"`
}

func (c *Config) Devices() ([]DeviceInfo, error) {
	var resp struct {
		Devices []DeviceInfo `json:"devices"`
	}
	if err := do(http.MethodGet, c.endpoint("/devices"), c.Token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Devices, nil
}

func (c *Config) Revoke(deviceID string) error {
	return do(http.MethodPost, c.endpoint("/devices/revoke"), c.Token, map[string]string{"device_id": deviceID}, nil)
}

type MemberInfo struct {
	AccountID string `json:"account_id"`
	Role      string `json:"role"`
}

func (c *Config) Members() ([]MemberInfo, error) {
	var resp struct {
		Members []MemberInfo `json:"members"`
	}
	if err := do(http.MethodGet, c.endpoint("/rooms/members"), c.Token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Members, nil
}

func (c *Config) Kick(accountID string) error {
	return do(http.MethodPost, c.endpoint("/rooms/kick"), c.Token, map[string]string{"account_id": accountID}, nil)
}
