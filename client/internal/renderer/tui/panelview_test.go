package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"statusphere-client/internal/presence"
	"statusphere-client/internal/stats"
)

func TestVOpensPanelPickerAndSwitches(t *testing.T) {
	m := chatModel()

	m = send(m, key("v"))
	if m.mode != modeView {
		t.Fatalf("v should open the panel picker, mode=%v", m.mode)
	}
	out := m.View()
	if !strings.Contains(out, "panel") || !strings.Contains(out, "Screen today") {
		t.Fatalf("picker should list panel views:\n%s", out)
	}

	m = send(m, key("down"))
	next, _ := m.runMenu()
	m = next.(model)
	if m.panel != panelBoard || m.mode != modeNone {
		t.Fatalf("selecting Screen today should switch the panel, panel=%v mode=%v", m.panel, m.mode)
	}

	// tab must not focus the board (nothing to type there)
	m = send(m, key("tab"))
	if m.focus != focusCards {
		t.Fatal("tab must be a no-op while the board view is active")
	}
}

func TestBoardPanelRendersLeaderboard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(stats.RoomScreen{Members: []stats.MemberScreen{
			{AccountID: "acc-bob", Seconds: 7200},
			{AccountID: meAccount, Seconds: 3600},
		}})
	}))
	defer srv.Close()

	m := chatModel()
	m.roomID = "room-1"
	m.room = stats.NewRoomScreenCache(srv.URL, "tok", "room-1")
	m.room.Prime("room-1")
	m.panel = panelBoard

	out := m.boardPanel(40, 16)
	for _, want := range []string{"screen today", "Bob", "2h0m", "1h0m"} {
		if !strings.Contains(out, want) {
			t.Fatalf("board panel missing %q:\n%s", want, out)
		}
	}
}

func TestBoardPanelSurvivesUnsortedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(stats.RoomScreen{Members: []stats.MemberScreen{
			{AccountID: "acc-a", Seconds: 60},
			{AccountID: "acc-b", Seconds: 6000},
		}})
	}))
	defer srv.Close()

	m := chatModel()
	m.roomID = "room-1"
	m.room = stats.NewRoomScreenCache(srv.URL, "tok", "room-1")
	m.room.Prime("room-1")

	out := m.boardPanel(40, 16)
	if !strings.Contains(out, "1h40m") {
		t.Fatalf("unsorted payload should still render every member:\n%s", out)
	}
}

func TestSparklineAxisMatchesWidth(t *testing.T) {
	h := &stats.Hourly{Hours: make([]int, 24)}
	h.Hours[0] = 600
	out := renderHourlySparkline(h)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("sparkline should be 3 lines, got %d", len(lines))
	}
	spark, axis := ansi.StringWidth(lines[1]), ansi.StringWidth(lines[2])
	if spark != 24 || axis != 24 {
		t.Fatalf("sparkline and axis must both be 24 cols, got %d and %d", spark, axis)
	}
}

func TestHourlySparkline(t *testing.T) {
	h := &stats.Hourly{Hours: make([]int, 24)}
	h.Hours[9] = 1800
	h.Hours[10] = 3600

	out := renderHourlySparkline(h)
	if !strings.Contains(out, "activity · today") || !strings.Contains(out, "█") {
		t.Fatalf("sparkline missing content:\n%s", out)
	}

	if renderHourlySparkline(&stats.Hourly{Hours: make([]int, 24)}) != "" {
		t.Fatal("all-zero day should render nothing")
	}
	if renderHourlySparkline(&stats.Hourly{Hours: []int{1, 2}}) != "" {
		t.Fatal("malformed hours should render nothing")
	}
}

func TestScreenModalIncludesSparkline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stats/hourly":
			h := stats.Hourly{DeviceID: "bob-1", Hours: make([]int, 24)}
			h.Hours[12] = 3600
			json.NewEncoder(w).Encode(h)
		default:
			json.NewEncoder(w).Encode(stats.Summary{Apps: []stats.AppStat{{App: "kitty", Seconds: 600}}})
		}
	}))
	defer srv.Close()

	m := chatModel()
	m.summary = stats.NewSummaryCache(srv.URL, "tok", "day", "room-1")
	m.hourly = stats.NewHourlyCache(srv.URL, "tok", "room-1")
	m.summary.Prime("bob-1")
	m.hourly.Prime("bob-1")
	selectAccount(&m, "acc-bob")
	m.mode = modeScreen

	out := m.View()
	for _, want := range []string{"screen time", "kitty", "activity · today"} {
		if !strings.Contains(out, want) {
			t.Fatalf("screen modal missing %q:\n%s", want, out)
		}
	}
}

func TestBoardShowsOfflineMemberName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(stats.RoomScreen{Members: []stats.MemberScreen{
			{AccountID: "acc-ann", Seconds: 900},
		}})
	}))
	defer srv.Close()

	m := chatModel()
	m.groups = append(m.groups, deviceGroup{key: "acc-ann", devices: []presence.Snapshot{
		{presence.KeyAccountID: "acc-ann", presence.KeyAccountName: "Ann", presence.KeyOffline: true},
	}})
	m.roomID = "room-1"
	m.room = stats.NewRoomScreenCache(srv.URL, "tok", "room-1")
	m.room.Prime("room-1")

	out := m.boardPanel(40, 16)
	if !strings.Contains(out, "Ann") {
		t.Fatalf("board should resolve names from the roster incl. offline members:\n%s", out)
	}
}
