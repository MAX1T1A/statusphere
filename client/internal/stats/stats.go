package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"statusphere-client/internal/transport"
)

type AppStat struct {
	App     string `json:"app"`
	Seconds int    `json:"seconds"`
}

type Summary struct {
	DeviceID string    `json:"device_id"`
	Period   string    `json:"period"`
	Since    string    `json:"since"`
	Apps     []AppStat `json:"apps"`
}

func Fetch(serverURL, token, period string) (*Summary, error) {
	u, _ := url.Parse(serverURL)
	u.Path = "/stats/summary"
	q := u.Query()
	q.Set("room_token", token)
	q.Set("device_id", transport.ID())
	q.Set("period", period)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var s Summary
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func Print(s *Summary) {
	fmt.Printf("Статистика за %s (с %s)\n\n", s.Period, s.Since)

	if len(s.Apps) == 0 {
		fmt.Println("Нет данных")
		return
	}

	for _, app := range s.Apps {
		h := app.Seconds / 3600
		m := (app.Seconds % 3600) / 60

		var time string
		if h > 0 {
			time = fmt.Sprintf("%d ч %d мин", h, m)
		} else {
			time = fmt.Sprintf("%d мин", m)
		}

		fmt.Printf("  %-20s %s\n", app.App, time)
	}
}
