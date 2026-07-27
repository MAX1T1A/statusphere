package stats

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Hourly struct {
	DeviceID string `json:"device_id"`
	Hours    []int  `json:"hours"`
}

type hourlyFetcher struct {
	room string
}

func (f hourlyFetcher) Path() string { return "/stats/hourly" }

func (f hourlyFetcher) Query(deviceID string) url.Values {
	q := url.Values{}
	q.Set("room", f.room)
	q.Set("device_id", deviceID)
	q.Set("tz_offset", strconv.Itoa(tzOffsetMinutes()))
	if name := tzName(); name != "" {
		q.Set("tz", name)
	}
	return q
}

func (f hourlyFetcher) New() any { return &Hourly{} }

func NewHourlyCache(serverURL, token, room string) *Cache {
	return NewCache(serverURL, token, hourlyFetcher{room: room})
}

func tzOffsetMinutes() int {
	_, sec := time.Now().Zone()
	return sec / 60
}

func tzName() string {
	if tz := os.Getenv("TZ"); strings.Contains(tz, "/") {
		return tz
	}
	if link, err := os.Readlink("/etc/localtime"); err == nil {
		if i := strings.Index(link, "zoneinfo/"); i >= 0 {
			return link[i+len("zoneinfo/"):]
		}
	}
	return ""
}
