package steam

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var procRoot = "/proc"

const (
	// USER_HZ. CGO is off so sysconf is unavailable, and it has been 100 on every
	// Linux port this client runs on.
	clockTicksPerSecond = 100
	cmdlineProbe        = 512
	environProbe        = 8192
)

type session struct {
	appID      int
	pid        int
	started    time.Time
	viaEnviron bool
}

// parseSteamLaunchAppID reads a NUL-separated cmdline. Steam's argv is
// "reaper SteamLaunch AppId=<id> -- <the game>", so anything past the "--" is the
// game's own and not ours to read.
func parseSteamLaunchAppID(cmdline []byte) int {
	seen := false
	for _, arg := range bytes.Split(cmdline, []byte{0}) {
		switch {
		case len(arg) == 0:
		case bytes.Equal(arg, []byte("SteamLaunch")):
			seen = true
		case !seen:
		case bytes.Equal(arg, []byte("--")):
			return 0
		default:
			if rest, ok := bytes.CutPrefix(arg, []byte("AppId=")); ok {
				id, err := strconv.Atoi(string(rest))
				if err != nil || id <= 0 {
					return 0
				}
				return id
			}
		}
	}
	return 0
}

// parseSteamAppIDEnv is the second signal, for a launcher we do not recognise.
// Steam's own helpers carry SteamAppId=0.
func parseSteamAppIDEnv(environ []byte) int {
	for _, kv := range bytes.Split(environ, []byte{0}) {
		rest, ok := bytes.CutPrefix(kv, []byte("SteamAppId="))
		if !ok {
			continue
		}
		id, err := strconv.Atoi(string(rest))
		if err != nil || id <= 0 {
			return 0
		}
		return id
	}
	return 0
}

// parseStartTicks returns field 22 of /proc/<pid>/stat. comm is field 2 and can
// hold spaces and parentheses, so the fields are counted from the last ')'.
func parseStartTicks(stat []byte) (int64, bool) {
	i := bytes.LastIndexByte(stat, ')')
	if i < 0 {
		return 0, false
	}
	fields := strings.Fields(string(stat[i+1:]))
	const startTime = 22 - 3 // the slice starts at field 3
	if len(fields) <= startTime {
		return 0, false
	}
	v, err := strconv.ParseInt(fields[startTime], 10, 64)
	return v, err == nil
}

func parseBootTime(procStat []byte) (int64, bool) {
	for _, line := range strings.Split(string(procStat), "\n") {
		rest, ok := strings.CutPrefix(line, "btime ")
		if !ok {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
		return v, err == nil
	}
	return 0, false
}

// Steam's runtimes and redistributables install like games and launch through the
// same reaper, so a scan of /proc finds them.
var nonGameAppIDs = map[int]bool{
	1070560: true, // Steam Linux Runtime 1.0 (scout)
	1391110: true, // Steam Linux Runtime 2.0 (soldier)
	1628350: true, // Steam Linux Runtime 3.0 (sniper)
	4183110: true, // Steam Linux Runtime 4.0
	1493710: true, // Proton Experimental
	2180100: true, // Proton Hotfix
	228980:  true, // Steamworks Common Redistributables
}

// There is a new Proton and a new runtime appid most years, and the name is the
// part that stays put. Proton is anchored on a word boundary: Protonwar is a game.
var nonGameName = regexp.MustCompile(`^(Proton( |$)|Steam Linux Runtime|Steamworks Common Redistributables|SteamVR|EasyAntiCheat Runtime|BattlEye Runtime)`)

func isNonGame(appID int, name string) bool {
	return nonGameAppIDs[appID] || (name != "" && nonGameName.MatchString(name))
}

// readInto fills buf with one read. /proc files are generated on read and stat as
// zero bytes, so os.ReadFile's stat-then-grow is wasted here.
func readInto(path string, buf []byte) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	return n, nil
}

func startedAt(pid int, boot time.Time) time.Time {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return time.Time{}
	}
	ticks, ok := parseStartTicks(data)
	if !ok || boot.IsZero() {
		return time.Time{}
	}
	return boot.Add(time.Duration(ticks) * time.Second / clockTicksPerSecond)
}

// Two games at once is rare but real - one still loading while another is up - and
// readdir order is not something to hang a card on, so the walk finishes and the
// newest session wins. Costs nothing in practice: with a game running the known pid
// is re-read instead, and with none the whole of /proc is walked either way.
func scanProc(name string, probe int, parse func([]byte) int, boot time.Time) (session, bool) {
	d, err := os.Open(procRoot)
	if err != nil {
		return session{}, false
	}
	defer d.Close()

	buf := make([]byte, probe)
	var best session
	found := false

	for {
		names, err := d.Readdirnames(256)
		if len(names) == 0 || err != nil {
			return best, found
		}
		for _, entry := range names {
			pid, err := strconv.Atoi(entry)
			if err != nil {
				continue
			}
			n, err := readInto(filepath.Join(procRoot, entry, name), buf)
			if err != nil || n == 0 {
				continue
			}
			appID := parse(buf[:n])
			if appID == 0 {
				continue
			}
			cur := session{
				appID:      appID,
				pid:        pid,
				started:    startedAt(pid, boot),
				viaEnviron: name == "environ",
			}
			if !found || cur.started.After(best.started) {
				best, found = cur, true
			}
		}
	}
}

// alive re-reads the one process already found instead of walking /proc again, and
// rejects a recycled pid: it has to still name the same app by the same signal.
func alive(s session) bool {
	name, parse, probe := "cmdline", parseSteamLaunchAppID, cmdlineProbe
	if s.viaEnviron {
		name, parse, probe = "environ", parseSteamAppIDEnv, environProbe
	}
	buf := make([]byte, probe)
	n, err := readInto(filepath.Join(procRoot, strconv.Itoa(s.pid), name), buf)
	return err == nil && parse(buf[:n]) == s.appID
}
