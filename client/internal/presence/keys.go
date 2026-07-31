package presence

const (
	KeyDeviceID     = "device_id"
	KeyAccountID    = "account_id"
	KeyAccountName  = "account_name"
	KeyDeviceName   = "device_name"
	KeyLastSeen     = "last_seen"
	KeyNudge        = "nudge_message"
	KeyCustomFields = "custom_fields"

	KeyRole          = "_role"
	KeyOffline       = "_offline"
	KeyIncognito     = "_incognito"
	KeyIncognitoNote = "_incognito_note"

	// KeyKind tells a machine from a person. A card for KindServer is read for
	// its metrics, not for what window is open on it.
	KeyKind = "_kind"

	// KeyHealth is ok/warn/crit as judged by the machine itself, absent while
	// nothing is wrong. KeyHealthNote names what went wrong, eg. "disk 92%".
	KeyHealth     = "_health"
	KeyHealthNote = "_health_note"

	KeyUptimeHours  = "uptime_hours"
	KeyCPUPercent   = "cpu_percent"
	KeyMemUsedMB    = "memory_used_mb"
	KeyMemTotalMB   = "memory_total_mb"
	KeyLoadAvg1m    = "load_avg_1m"
	KeyPackageCount = "package_count"
	KeyCPUCount     = "cpu_count"

	KeyDiskUsedPercent = "disk_used_percent"
	KeyDiskFreeGB      = "disk_free_gb"

	KeyActiveApp       = "active_app"
	KeyActiveWindow    = "active_window"
	KeyActiveWorkspace = "active_workspace"

	// KeyMusic is whatever mpris player is playing, browsers included.
	KeyMusic = "music"

	KeySpotifyStatus   = "spotify_status"
	KeySpotifyTrack    = "spotify_track"
	KeySpotifyArtist   = "spotify_artist"
	KeySpotifyAlbum    = "spotify_album"
	KeySpotifyArtURL   = "spotify_art_url"
	KeySpotifyURI      = "spotify_uri"
	KeySpotifyDisplay  = "spotify_display"
	KeySpotifyPosition = "spotify_position"
	KeySpotifyLength   = "spotify_length"
)

// Device kinds. An empty kind reads as KindDesktop - that is what every client
// that predates this key sends.
const (
	KindDesktop = "desktop"
	KindServer  = "server"
)

func (s Snapshot) Kind() string {
	if kind := s.String(KeyKind); kind != "" {
		return kind
	}
	return KindDesktop
}

func (s Snapshot) IsServer() bool { return s.Kind() == KindServer }
