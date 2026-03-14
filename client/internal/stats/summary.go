package stats

import (
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
