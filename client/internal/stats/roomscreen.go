package stats

import (
	"net/url"
	"strconv"
)

type MemberScreen struct {
	AccountID string `json:"account_id"`
	Seconds   int    `json:"seconds"`
}

type RoomScreen struct {
	Members []MemberScreen `json:"members"`
}

type roomScreenFetcher struct {
	room string
}

func (f roomScreenFetcher) Path() string { return "/stats/room" }

func (f roomScreenFetcher) Query(string) url.Values {
	q := url.Values{}
	q.Set("room", f.room)
	q.Set("tz_offset", strconv.Itoa(tzOffsetMinutes()))
	if name := tzName(); name != "" {
		q.Set("tz", name)
	}
	return q
}

func (f roomScreenFetcher) New() any { return &RoomScreen{} }

func NewRoomScreenCache(serverURL, token, room string) *Cache {
	return NewCache(serverURL, token, roomScreenFetcher{room: room})
}
