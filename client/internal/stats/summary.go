package stats

import (
	"fmt"
	"net/url"
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

type summaryFetcher struct {
	period string
}

func (f summaryFetcher) Path() string { return "/stats/summary" }

func (f summaryFetcher) Query(deviceID string) url.Values {
	q := url.Values{}
	q.Set("device_id", deviceID)
	q.Set("period", f.period)
	return q
}

func (f summaryFetcher) New() any { return &Summary{} }

func NewSummaryCache(serverURL, token, period string) *Cache {
	return NewCache(serverURL, token, summaryFetcher{period: period})
}

func PrintSummary(s *Summary) {
	fmt.Printf("Статистика за %s (с %s)\n\n", s.Period, s.Since)

	if len(s.Apps) == 0 {
		fmt.Println("Нет данных")
		return
	}

	for _, app := range s.Apps {
		h := app.Seconds / 3600
		m := (app.Seconds % 3600) / 60

		var t string
		if h > 0 {
			t = fmt.Sprintf("%d ч %d мин", h, m)
		} else {
			t = fmt.Sprintf("%d мин", m)
		}

		fmt.Printf("  %-20s %s\n", app.App, t)
	}
}
