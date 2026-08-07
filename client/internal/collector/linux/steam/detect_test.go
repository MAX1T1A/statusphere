package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func nul(args ...string) []byte {
	out := []byte{}
	for _, a := range args {
		out = append(out, a...)
		out = append(out, 0)
	}
	return out
}

func TestParseSteamLaunchAppID(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want int
	}{
		{"reaper", []string{"/steam/ubuntu12_32/reaper", "SteamLaunch", "AppId=1174180", "--", "/games/RDR2.exe"}, 1174180},
		{"steamwebhelper", []string{"/steam/ubuntu12_64/steamwebhelper", "-nocrashdialog", "-steampid=3744005"}, 0},
		{"zero appid", []string{"reaper", "SteamLaunch", "AppId=0", "--"}, 0},
		{"garbage appid", []string{"reaper", "SteamLaunch", "AppId=nope"}, 0},
		{"appid without the launch marker", []string{"something", "AppId=1174180"}, 0},
		{"the game's own args are not ours", []string{"reaper", "SteamLaunch", "--", "AppId=999"}, 0},
		{"empty", nil, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseSteamLaunchAppID(nul(c.argv...)); got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestParseSteamAppIDEnv(t *testing.T) {
	cases := []struct {
		name string
		env  []string
		want int
	}{
		{"a game", []string{"LANG=en_US.UTF-8", "SteamAppId=2183900", "SteamGameId=2183900"}, 2183900},
		{"steam's own helper", []string{"SteamAppId=0", "STEAMSCRIPT=/usr/lib/steam/steam"}, 0},
		{"a longer key that starts the same", []string{"SteamAppUser=berupor"}, 0},
		{"nothing", []string{"HOME=/home/x"}, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseSteamAppIDEnv(nul(c.env...)); got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestParseStartTicks(t *testing.T) {
	// A real line, captured from this machine
	const real = `3855448 (cat) R 3855447 3855448 3743063 34816 3855448 4194304 90 0 0 0 0 0 0 0 20 0 1 0 70132900 8720384 271 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 4 0 0 0 0 0 0 0 0 0 0 0 0 0`

	if got, ok := parseStartTicks([]byte(real)); !ok || got != 70132900 {
		t.Errorf("got %d ok=%v, want 70132900", got, ok)
	}

	// comm is arbitrary bytes: spaces and parentheses inside it must not shift the count
	const nasty = `42 (a ) weird ) name) S 1 42 42 0 -1 4194304 0 0 0 0 0 0 0 0 20 0 1 0 12345 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0`
	if got, ok := parseStartTicks([]byte(nasty)); !ok || got != 12345 {
		t.Errorf("got %d ok=%v, want 12345", got, ok)
	}

	for name, in := range map[string]string{
		"empty":      ``,
		"no comm":    `42 S 1 2 3`,
		"too short":  `42 (cat) S 1 2 3`,
		"unparsable": `42 (cat) S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 x 0`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := parseStartTicks([]byte(in)); ok {
				t.Error("claimed success on junk")
			}
		})
	}
}

func TestParseBootTime(t *testing.T) {
	const procStat = `cpu  1 2 3 4
cpu0 1 2 3 4
intr 100 0 0
ctxt 500
btime 1785411463
processes 900
`
	if got, ok := parseBootTime([]byte(procStat)); !ok || got != 1785411463 {
		t.Errorf("got %d ok=%v", got, ok)
	}
	if _, ok := parseBootTime([]byte("cpu 1 2 3\n")); ok {
		t.Error("claimed a boot time that was not there")
	}
}

func TestIsNonGame(t *testing.T) {
	cases := []struct {
		appID int
		name  string
		want  bool
	}{
		{1070560, "", true},       // by appid, before the name is known
		{0, "Proton 9.0-4", true}, // by name, for next year's build
		{0, "Steam Linux Runtime 3.0 (sniper)", true},
		{0, "Steamworks Common Redistributables", true},
		{0, "Protonwar", false}, // a real title, not a compat layer
		{1174180, "Red Dead Redemption 2", false},
	}

	for _, c := range cases {
		if got := isNonGame(c.appID, c.name); got != c.want {
			t.Errorf("isNonGame(%d, %q) = %v, want %v", c.appID, c.name, got, c.want)
		}
	}
}

// fakeProc lays out a /proc with one directory per pid: the cmdline the scan
// parses, and a stat whose field 22 sets when that process started.
func fakeProc(t *testing.T, procs map[int][2]string) string {
	t.Helper()
	root := t.TempDir()
	for pid, p := range procs {
		dir := filepath.Join(root, strconv.Itoa(pid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		write := func(name, body string) {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		write("cmdline", p[0])
		// pid, comm, then the state and eighteen fields before starttime
		write("stat", fmt.Sprintf("%d (game) S%s %s", pid, strings.Repeat(" 0", 18), p[1]))
	}
	// Things that are not processes, and a process that is not a game
	if err := os.WriteFile(filepath.Join(root, "uptime"), []byte("1 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestScanProcPrefersTheNewestSession(t *testing.T) {
	old := procRoot
	t.Cleanup(func() { procRoot = old })

	procRoot = fakeProc(t, map[int][2]string{
		41: {string(nul("reaper", "SteamLaunch", "AppId=1174180", "--", "/games/a")), "1000"},
		42: {string(nul("/usr/lib/firefox/firefox")), "2000"},
		43: {string(nul("reaper", "SteamLaunch", "AppId=413150", "--", "/games/b")), "3000"},
	})

	boot := time.Unix(1785411463, 0)
	got, ok := scanProc("cmdline", cmdlineProbe, parseSteamLaunchAppID, boot)
	if !ok {
		t.Fatal("found no game at all")
	}
	if got.appID != 413150 {
		t.Errorf("appid = %d, want the one started last", got.appID)
	}
	if want := boot.Add(30 * time.Second); !got.started.Equal(want) {
		t.Errorf("started = %v, want %v", got.started, want)
	}
}

func TestScanProcFindsNothing(t *testing.T) {
	old := procRoot
	t.Cleanup(func() { procRoot = old })

	procRoot = fakeProc(t, map[int][2]string{
		42: {string(nul("/usr/lib/firefox/firefox")), "2000"},
	})
	if _, ok := scanProc("cmdline", cmdlineProbe, parseSteamLaunchAppID, time.Now()); ok {
		t.Error("a browser was taken for a game")
	}
}
