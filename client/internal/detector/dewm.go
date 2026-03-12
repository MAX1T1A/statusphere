package detector

import (
	"os"
	"strings"
)

var envChecks = []struct {
	env  string
	name string
}{
	{"HYPRLAND_INSTANCE_SIGNATURE", "hyprland"},
	{"SWAYSOCK", "sway"},
	{"GNOME_DESKTOP_SESSION_ID", "gnome"},
	{"KDE_FULL_SESSION", "kde"},
}

func detectDEWM() string {
	for _, c := range envChecks {
		if os.Getenv(c.env) != "" {
			return c.name
		}
	}

	if xdg := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP")); xdg != "" {
		return xdg
	}

	return ""
}
