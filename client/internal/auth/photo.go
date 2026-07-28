package auth

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const MaxUploadBytes = 8 * 1024 * 1024

type PhotoInfo struct {
	AccountID string `json:"account_id"`
	PhotoID   string `json:"photo_id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

// PostPhoto shares path as this account's current photo status, replacing any
// previous one. The server re-encodes/resizes it, so no client-side work beyond
// a size cap is done here.
func (c *Config) PostPhoto(path string) (*PhotoInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxUploadBytes {
		return nil, fmt.Errorf("photo too large: %d bytes (max %d)", len(data), MaxUploadBytes)
	}

	var resp PhotoInfo
	body := map[string]string{"room": c.RoomID, "image_base64": base64.StdEncoding.EncodeToString(data)}
	if err := do(http.MethodPost, c.endpoint("/photos"), c.Token, body, &resp); err != nil {
		return nil, fmt.Errorf("post photo: %w", err)
	}
	return &resp, nil
}

func (c *Config) ListRoomPhotos() ([]PhotoInfo, error) {
	var resp struct {
		Photos []PhotoInfo `json:"photos"`
	}
	endpoint := c.endpoint("/photos") + "?room=" + url.QueryEscape(c.RoomID)
	if err := do(http.MethodGet, endpoint, c.Token, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Photos, nil
}

// FetchPhotoBlob downloads a room member's current photo bytes. Used internally
// by the always-running "--ui json" process, which caches the result to local
// disk since Quickshell's Image element can't attach the X-Room-Token header.
func FetchPhotoBlob(serverURL, token, roomID, accountID, photoID string) ([]byte, error) {
	u := strings.TrimRight(serverURL, "/") + "/photos/" + accountID + "/" + photoID +
		"?room=" + url.QueryEscape(roomID)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Room-Token", token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("photo blob: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
