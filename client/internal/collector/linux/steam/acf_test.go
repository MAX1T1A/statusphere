package steam

import "testing"

const manifest = `"AppState"
{
	"appid"		"1174180"
	"Universe"		"1"
	"name"		"Red Dead Redemption 2"
	"StateFlags"		"4"
	"installdir"		"Red Dead Redemption 2"
	"LastPlayed"		"1782741652"
	"InstalledDepots"
	{
		"1174182"
		{
			"manifest"		"2258488483491377476"
			"name"		"a depot that also calls its key name"
		}
	}
	"UserConfig"
	{
		"language"		"russian"
	}
}
`

func TestParseFlatTakesTheRootObjectOnly(t *testing.T) {
	got := parseFlat([]byte(manifest))

	if got["name"] != "Red Dead Redemption 2" {
		t.Errorf("name = %q, a nested key won over AppState's", got["name"])
	}
	if got["appid"] != "1174180" {
		t.Errorf("appid = %q", got["appid"])
	}
	if got["lastplayed"] != "1782741652" {
		t.Errorf("keys should be lower-cased, got %v", got)
	}
	if _, ok := got["manifest"]; ok {
		t.Error("a depot's key leaked into the root scalars")
	}
}

func TestParseFlatEscapes(t *testing.T) {
	got := parseFlat([]byte(`"AppState" { "name" "Tom Clancy's \"Test\"" }`))
	if got["name"] != `Tom Clancy's "Test"` {
		t.Errorf("name = %q", got["name"])
	}
}

func TestParseFlatSurvivesJunk(t *testing.T) {
	for name, in := range map[string]string{
		"empty":        ``,
		"truncated":    `"AppState" { "name" "Red Dead`,
		"stray brace":  `} "name" "x"`,
		"no root":      `"name" "x"`,
		"only comment": "// nothing here\n",
	} {
		t.Run(name, func(t *testing.T) {
			parseFlat([]byte(in)) // must not panic
		})
	}
}

func TestLibraryPathsSkipsTheAppsBlock(t *testing.T) {
	const vdf = `"libraryfolders"
{
	"0"
	{
		"path"		"/home/evgeniy/.local/share/Steam"
		"apps"
		{
			"1174180"		"128281007399"
			"path"		"a decoy one level too deep"
		}
	}
	"1"
	{
		"path"		"/mnt/games/SteamLibrary"
	}
}
`
	got := libraryPaths([]byte(vdf))
	want := []string{"/home/evgeniy/.local/share/Steam", "/mnt/games/SteamLibrary"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("root %d = %q, want %q", i, got[i], want[i])
		}
	}
}
