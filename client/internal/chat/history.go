package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Message struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
	At   string `json:"at"`
}

func FetchHistory(serverURL, token, roomID string) ([]Message, error) {
	u, err := url.Parse(strings.TrimRight(serverURL, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = "/messages"
	u.RawQuery = url.Values{"room": {roomID}}.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Room-Token", token)

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("history: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var out struct {
		Messages []Message `json:"messages"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Messages, nil
}
