package utils

import "strings"

var knownApps = map[string]string{
	"brave-browser":          "Brave",
	"google-chrome":          "Chrome",
	"chromium":               "Chromium",
	"firefox":                "Firefox",
	"org.mozilla.firefox":    "Firefox",
	"telegram-desktop":       "Telegram",
	"org.telegram.desktop":   "Telegram",
	"discord":                "Discord",
	"com.discordapp.Discord": "Discord",
	"spotify":                "Spotify",
	"com.spotify.Client":     "Spotify",
	"code":                   "VS Code",
	"Code":                   "VS Code",
	"obsidian":               "Obsidian",
	"kitty":                  "Kitty",
	"alacritty":              "Alacritty",
	"org.wezfurlong.wezterm": "WezTerm",
	"steam":                  "Steam",
	"thunar":                 "Thunar",
	"nautilus":               "Files",
	"org.gnome.Nautilus":     "Files",
	"vlc":                    "VLC",
	"mpv":                    "mpv",
	"gimp":                   "GIMP",
	"blender":                "Blender",
	"libreoffice":            "LibreOffice",
	"obs":                    "OBS",
	"com.obsproject.Studio":  "OBS",
	"slack":                  "Slack",
	"com.slack.Slack":        "Slack",
}

func CleanAppName(raw string) string {
	if name, ok := knownApps[raw]; ok {
		return name
	}

	for prefix, name := range knownApps {
		if strings.HasPrefix(raw, prefix) {
			return name
		}
	}

	cleaned := raw
	if idx := strings.LastIndex(cleaned, "."); idx != -1 {
		last := cleaned[idx+1:]
		if len(last) < 40 {
			cleaned = last
		}
	}

	cleaned = strings.TrimPrefix(cleaned, "org.")
	cleaned = strings.TrimPrefix(cleaned, "com.")
	cleaned = strings.TrimPrefix(cleaned, "io.")
	cleaned = strings.TrimPrefix(cleaned, "net.")

	if idx := strings.Index(cleaned, "_"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	return cleaned
}
