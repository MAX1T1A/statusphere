package steam

import (
	"os"
	"path/filepath"
	"strconv"
)

type app struct {
	appID int
	name  string
}

// Most likely first. ~/.steam/steam is usually a symlink to the first one; the
// flatpak path is empty unless Steam was installed that way.
func steamRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", "data", "Steam"),
	}
}

func findRoot() string {
	for _, root := range steamRoots() {
		if fi, err := os.Stat(filepath.Join(root, "steamapps")); err == nil && fi.IsDir() {
			return root
		}
	}
	return ""
}

// A game lives in whichever library it was installed into, not necessarily the
// default one.
func libraryRoots(root string) []string {
	def := filepath.Join(root, "steamapps")
	out := []string{def}

	data, err := os.ReadFile(filepath.Join(def, "libraryfolders.vdf"))
	if err != nil {
		return out
	}
	for _, path := range libraryPaths(data) {
		lib := filepath.Join(path, "steamapps")
		if lib == def {
			continue
		}
		out = append(out, lib)
	}
	return out
}

// The file name follows from the appid, so this is at most one failed open per
// library rather than a directory listing.
func findApp(libs []string, appID int) (app, bool) {
	name := "appmanifest_" + strconv.Itoa(appID) + ".acf"
	for _, lib := range libs {
		data, err := os.ReadFile(filepath.Join(lib, name))
		if err != nil {
			continue
		}
		return app{appID: appID, name: parseFlat(data)["name"]}, true
	}
	return app{}, false
}
